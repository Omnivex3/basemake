package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

var initDemo bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up basemake with a guided wizard",
	Long: `Interactive setup that detects your database, helps you pick
an AI provider, and runs a test query — all in one go.

  basemake init              # Guided wizard
  basemake init --demo       # Start with demo data (no real DB needed)
  basemake init --provider openrouter --key sk-...  # Non-interactive

The fastest way to go from zero to "wow that just worked."`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initDemo, "demo", false, "Initialize with demo data (no real database required)")
	initCmd.Flags().String("provider", "", "AI provider (openrouter, groq, deepseek, openai, anthropic, ollama)")
	initCmd.Flags().String("key", "", "API key for the AI provider")
	initCmd.Flags().String("db", "", "Database DSN (e.g. postgres://user:pass@localhost/db)")
}

// ── Wizard ──

func runInit(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	blue := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	reset := "\033[0m"
	bold := "\033[1m"
	dim := "\033[2m"

	// ── Welcome ──
	fmt.Printf("%s┌──────────────────────────────────────────┐%s\n", blue, reset)
	fmt.Printf("%s│  %sWelcome to basemake 🚀%s                 %s│%s\n", blue, bold, reset, blue, reset)
	fmt.Printf("%s│  Let's get you up and running.            │%s\n", blue, reset)
	fmt.Printf("%s└──────────────────────────────────────────┘%s\n", blue, reset)
	fmt.Println()

	// ── Step 1: Database ──
	fmt.Printf("%sStep 1: Database%s\n", bold, reset)
	fmt.Println()

	var databaseConn db.Database
	var dbName string
	var dsn string

	// Check for --db flag first (non-interactive)
	if flagDB, _ := cmd.Flags().GetString("db"); flagDB != "" {
		conn, err := db.Connect(flagDB)
		if err != nil {
			return fmt.Errorf("connect to database: %w", err)
		}
		databaseConn = conn
		dbName = conn.Name()
		dsn = flagDB
		fmt.Printf("  %s✓ Connected to %s%s\n", green, dbName, reset)
	} else {
		databaseConn, dbName, dsn = selectDatabase(reader, green, yellow, dim, bold, reset)
	}

	if databaseConn != nil {
		// Introspect schema
		schema, err := databaseConn.Introspect(cmd.Context())
		if err == nil {
			_ = db.SaveSchema(schema)
		}
		if dsn != "" {
			_ = db.SaveDSN(dsn)
		}
		fmt.Printf("  %s✅ Connected to %s%s\n", green, dbName, reset)
		fmt.Printf("  %s  Schema cached: %d tables%s\n", dim, len(schema.Tables), reset)
		fmt.Println()
	} else {
		fmt.Printf("  %s⏭ No database configured. You can connect later with `basemake connect`%s\n", dim, reset)
		fmt.Println()
	}

	// ── Step 2: AI Provider ──
	fmt.Printf("%sStep 2: AI Provider%s\n", bold, reset)
	fmt.Println()

	providerFlag, _ := cmd.Flags().GetString("provider")
	keyFlag, _ := cmd.Flags().GetString("key")

	if providerFlag != "" && keyFlag != "" {
		// Non-interactive mode
		saveProviderConfig(providerFlag, keyFlag)
		fmt.Printf("  %s✅ Configured %s%s\n", green, providerFlag, reset)
	} else {
		fmt.Printf("  %sWhich AI provider do you want to use?%s\n", bold, reset)
		fmt.Println()
		fmt.Printf("  %s1) OpenRouter%s  %s(free tier — deepseek-chat, ~$0.14/M tokens)%s\n", bold, reset, dim, reset)
		fmt.Printf("  %s2) Groq%s        %s(free, fast — llama3-70b)%s\n", bold, reset, dim, reset)
		fmt.Printf("  %s3) DeepSeek%s     %s(cheapest — $0.14/M input)%s\n", bold, reset, dim, reset)
		fmt.Printf("  %s4) OpenAI%s       %s(most compatible — requires paid key)%s\n", bold, reset, dim, reset)
		fmt.Printf("  %s5) Anthropic%s    %s(best for complex queries)%s\n", bold, reset, dim, reset)
		fmt.Printf("  %s6) Ollama%s       %s(local — runs on your machine)%s\n", bold, reset, dim, reset)
		fmt.Printf("  %ss) Skip%s         %s(use basemake in SQL-only mode)%s\n", bold, reset, dim, reset)
		fmt.Print("\n  → [1-6/s]: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		providerName, providerModel, baseURL := resolveProvider(choice)

		if providerName == "" || choice == "s" {
			fmt.Printf("  %s⏭ AI provider skipped. Configure later with `basemake config set ai_provider <name>`%s\n", dim, reset)
		} else {
			if keyFlag != "" {
				saveProviderConfigWithKey(providerName, providerModel, baseURL, keyFlag)
			} else {
				// Prompt for API key
				fmt.Printf("\n  %sEnter your %s API key:%s\n", bold, providerName, reset)
				fmt.Printf("  %s  (paste your key — input is hidden)%s\n", dim, reset)
				fmt.Print("  → ")
				keyBytes, _ := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				key := strings.TrimSpace(string(keyBytes))

				if key == "" {
					fmt.Printf("  %s⏭ No key entered. You can set it later with `basemake config set ...`%s\n", dim, reset)
				} else {
					saveProviderConfigWithKey(providerName, providerModel, baseURL, key)
					fmt.Printf("  %s✅ API key saved for %s%s\n", green, providerName, reset)

					// Quick validation
					if validateKey(providerName, key, baseURL) {
						fmt.Printf("  %s  ✓ Key validated successfully%s\n", green, reset)
					} else {
						fmt.Printf("  %s  ⚠ Could not validate key (provider may be unreachable)%s\n", yellow, reset)
					}
				}
			}
		}
	}

	fmt.Println()

	// ── Step 3: Test Drive ──
	fmt.Printf("%sStep 3: Test Drive%s\n", bold, reset)
	fmt.Println()

	if databaseConn != nil {
		fmt.Printf("  %sRunning: \"show me tables in the database\"%s\n", dim, reset)
		time.Sleep(500 * time.Millisecond) // Brief pause for suspense

		// Try a simple query
		rows, err := databaseConn.Query(cmd.Context(), "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name LIMIT 10")
		if err != nil {
			// Fallback for PostgreSQL/MySQL
			rows, err = databaseConn.Query(cmd.Context(),
				"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name LIMIT 10")
			if err != nil {
				fmt.Printf("  %s  ⚠ Test query failed: %s%s\n", yellow, friendlyDBError(err), reset)
				fmt.Printf("  %s  Your connection works — try a query manually%s\n", dim, reset)
			} else {
				printTestResults(rows)
			}
		} else {
			printTestResults(rows)
		}
	} else {
		fmt.Printf("  %sNo database connected, skipping test.%s\n", dim, reset)
		fmt.Printf("  %sRun `basemake connect postgres://...` when you're ready.%s\n", dim, reset)
	}

	fmt.Println()
	fmt.Printf("%s┌──────────────────────────────────────────┐%s\n", green, reset)
	fmt.Printf("%s│  %sYou're all set! 🎉%s                       %s│%s\n", green, bold, reset, green, reset)
	fmt.Printf("%s│                                          │%s\n", green, reset)
	fmt.Printf("%s│  %sNext steps:%s                             %s│%s\n", green, bold, reset, green, reset)
	fmt.Printf("%s│                                          │%s\n", green, reset)
	fmt.Printf("%s│  basemake \"show me ...\"%s                   │%s\n", green, dim, reset)
	fmt.Printf("%s│  basemake          (interactive mode)%s    │%s\n", green, dim, reset)
	fmt.Printf("%s│  basemake doctor   (check everything)%s    │%s\n", green, dim, reset)
	fmt.Printf("%s└──────────────────────────────────────────┘%s\n", green, reset)
	fmt.Println()

	return nil
}

// ── Database Detection ──

type detectedDB struct {
	label string
	dsn   string
}

func detectDatabases() []detectedDB {
	var results []detectedDB

	// Check common ports
	checks := []struct {
		port    int
		name    string
		dsnTmpl string
	}{
		{5432, "PostgreSQL", "postgres://postgres@localhost:5432/postgres?sslmode=disable"},
		{5433, "PostgreSQL (alt)", "postgres://postgres@localhost:5433/postgres?sslmode=disable"},
		{3306, "MySQL", "mysql://root@tcp(localhost:3306)/"},
	}

	for _, c := range checks {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", c.port), 2*time.Second)
		if err == nil {
			conn.Close()
			results = append(results, detectedDB{
				label: fmt.Sprintf("%s running on localhost:%d", c.name, c.port),
				dsn:   c.dsnTmpl,
			})
		}
	}

	// Check Docker containers running databases
	if dockerDBs := detectDockerDBs(); dockerDBs != nil {
		results = append(results, dockerDBs...)
	}

	return results
}

func detectDockerDBs() []detectedDB {
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Image}}\t{{.Ports}}").Output()
	if err != nil {
		return nil
	}

	var results []detectedDB
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		name := parts[0]
		image := strings.ToLower(parts[1])
		ports := parts[2]

		if strings.Contains(image, "postgres") {
			// Extract host port
			port := extractPort(ports, "5432")
			results = append(results, detectedDB{
				label: fmt.Sprintf("PostgreSQL in Docker (%s) on localhost:%s", name, port),
				dsn:   fmt.Sprintf("postgres://postgres@127.0.0.1:%s/postgres?sslmode=disable", port),
			})
		}
		if strings.Contains(image, "mysql") || strings.Contains(image, "mariadb") {
			port := extractPort(ports, "3306")
			results = append(results, detectedDB{
				label: fmt.Sprintf("MySQL in Docker (%s) on localhost:%s", name, port),
				dsn:   fmt.Sprintf("mysql://root@tcp(127.0.0.1:%s)/", port),
			})
		}
	}
	return results
}

func extractPort(ports, defaultPort string) string {
	// Port format: "0.0.0.0:5432->5432/tcp"
	parts := strings.Split(ports, "->")
	if len(parts) > 0 {
		hostPart := parts[0]
		idx := strings.LastIndex(hostPart, ":")
		if idx >= 0 {
			p := hostPart[idx+1:]
			// Trim any trailing content
			p = strings.TrimSpace(p)
			if p != "" {
				return p
			}
		}
	}
	return defaultPort
}

// ── Demo Database ──

const demoDBSQL = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    price REAL NOT NULL,
    category TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    total REAL NOT NULL,
    ordered_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

DELETE FROM orders;
DELETE FROM products;
DELETE FROM users;

INSERT INTO users VALUES (1, 'Alice M.', 'alice@example.com', '2024-01-15');
INSERT INTO users VALUES (2, 'Bob K.', 'bob@example.com', '2024-02-20');
INSERT INTO users VALUES (3, 'Charlie D.', 'charlie@example.com', '2024-03-10');
INSERT INTO users VALUES (4, 'Diana P.', 'diana@example.com', '2024-04-05');
INSERT INTO users VALUES (5, 'Eve R.', 'eve@example.com', '2024-05-12');
INSERT INTO users VALUES (6, 'Frank Z.', 'frank@example.com', '2024-06-01');
INSERT INTO users VALUES (7, 'Grace W.', 'grace@example.com', '2024-06-15');
INSERT INTO users VALUES (8, 'Henry L.', 'henry@example.com', '2024-07-01');
INSERT INTO users VALUES (9, 'Ivy N.', 'ivy@example.com', '2024-07-20');
INSERT INTO users VALUES (10, 'Jack S.', 'jack@example.com', '2024-08-05');

INSERT INTO products VALUES (1, 'Wireless Mouse', 29.99, 'Electronics');
INSERT INTO products VALUES (2, 'Mechanical Keyboard', 89.99, 'Electronics');
INSERT INTO products VALUES (3, 'USB-C Hub', 34.99, 'Accessories');
INSERT INTO products VALUES (4, '27" Monitor', 299.99, 'Electronics');
INSERT INTO products VALUES (5, 'Webcam HD', 79.99, 'Electronics');
INSERT INTO products VALUES (6, 'Standing Desk', 499.99, 'Furniture');
INSERT INTO products VALUES (7, 'Ergonomic Chair', 349.99, 'Furniture');
INSERT INTO products VALUES (8, 'Noise Cancelling Headphones', 149.99, 'Audio');
INSERT INTO products VALUES (9, 'Laptop Stand', 39.99, 'Accessories');
INSERT INTO products VALUES (10, 'Desk Lamp', 24.99, 'Furniture');

INSERT INTO orders VALUES (1, 1, 1, 2, 59.98, '2024-06-10');
INSERT INTO orders VALUES (2, 1, 3, 1, 34.99, '2024-06-10');
INSERT INTO orders VALUES (3, 2, 4, 1, 299.99, '2024-06-12');
INSERT INTO orders VALUES (4, 3, 2, 1, 89.99, '2024-06-15');
INSERT INTO orders VALUES (5, 4, 8, 1, 149.99, '2024-06-20');
INSERT INTO orders VALUES (6, 5, 6, 1, 499.99, '2024-07-01');
INSERT INTO orders VALUES (7, 1, 9, 2, 79.98, '2024-07-05');
INSERT INTO orders VALUES (8, 6, 5, 1, 79.99, '2024-07-10');
INSERT INTO orders VALUES (9, 7, 7, 1, 349.99, '2024-07-15');
INSERT INTO orders VALUES (10, 8, 10, 2, 49.98, '2024-07-20');
INSERT INTO orders VALUES (11, 9, 1, 1, 29.99, '2024-08-01');
INSERT INTO orders VALUES (12, 10, 4, 1, 299.99, '2024-08-05');
INSERT INTO orders VALUES (13, 2, 8, 1, 149.99, '2024-08-10');
INSERT INTO orders VALUES (14, 3, 3, 3, 104.97, '2024-08-15');
INSERT INTO orders VALUES (15, 5, 2, 1, 89.99, '2024-08-20');
`

func createDemoDB() (db.Database, string, string, error) {
	home, _ := os.UserHomeDir()
	demoDir := filepath.Join(home, ".basemake")
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		return nil, "", "", fmt.Errorf("create demo dir: %w", err)
	}

	dbPath := filepath.Join(demoDir, "demo.db")
	dsn := "sqlite:" + dbPath

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("open demo db: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, "", "", fmt.Errorf("wal: %w", err)
	}

	for _, stmt := range strings.Split(demoDBSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := conn.Exec(stmt); err != nil {
			return nil, "", "", fmt.Errorf("demo seed: %w\nSQL: %s", err, stmt[:min(len(stmt), 80)])
		}
	}

	// Connect via basemake's db layer
	database, err := db.Connect(dsn)
	if err != nil {
		return nil, "", "", fmt.Errorf("connect demo: %w", err)
	}

	return database, fmt.Sprintf("Demo DB (users, products, orders)"), dsn, nil
}

// ── Provider Resolution ──

func resolveProvider(choice string) (name, model, baseURL string) {
	switch strings.TrimSpace(choice) {
	case "1", "openrouter":
		return "OpenRouter", "deepseek/deepseek-chat", "https://openrouter.ai/api/v1"
	case "2", "groq":
		return "Groq", "llama3-70b-8192", "https://api.groq.com/openai/v1"
	case "3", "deepseek":
		return "DeepSeek", "deepseek-chat", "https://api.deepseek.com/v1"
	case "4", "openai":
		return "OpenAI", "gpt-4", "https://api.openai.com/v1"
	case "5", "anthropic":
		return "Anthropic", "claude-sonnet-4-20250514", "https://api.anthropic.com"
	case "6", "ollama":
		return "Ollama", "llama3", "http://localhost:11434/v1"
	default:
		return "", "", ""
	}
}

func saveProviderConfig(provider, key string) {
	name, model, baseURL := resolveProvider(provider)
	if name == "" {
		return
	}
	saveProviderConfigWithKey(name, model, baseURL, key)
}

func saveProviderConfigWithKey(name, model, baseURL, key string) {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	switch name {
	case "OpenRouter", "Groq", "DeepSeek", "OpenAI":
		cfg.AIProvider = "openai"
		cfg.OpenAIModel = model
		cfg.OpenAIBaseURL = baseURL
		// Save key to env file
		_ = saveKeyToEnv("OPENAI_API_KEY", key)
	case "Anthropic":
		cfg.AIProvider = "anthropic"
		cfg.AnthropicModel = model
		cfg.AnthropicBaseURL = baseURL
		_ = saveKeyToEnv("ANTHROPIC_API_KEY", key)
	case "Ollama":
		cfg.AIProvider = "ollama"
		cfg.OllamaModel = model
		cfg.OllamaBaseURL = baseURL
		// No key needed for Ollama
	}

	// Also set AI_PROVIDER env var for immediate use
	os.Setenv("AI_PROVIDER", cfg.AIProvider)

	_ = cfg.Save()
}

func saveKeyToEnv(varName, key string) error {
	home, _ := os.UserHomeDir()
	envPath := filepath.Join(home, ".basemake", "env")

	// Read existing env to preserve other vars
	existing := make(map[string]string)
	if data, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
			if len(parts) == 2 {
				existing[parts[0]] = parts[1]
			}
		}
	}

	// Set or update
	existing[varName] = key
	if varName == "OPENAI_API_KEY" {
		existing["AI_PROVIDER"] = "openai"
	}

	// Write back
	var sb strings.Builder
	for k, v := range existing {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}

	dir := filepath.Dir(envPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(envPath, []byte(sb.String()), 0600)
}

// ── Key Validation ──

func validateKey(provider, key, baseURL string) bool {
	// Quick reachability check — lightweight, non-blocking
	host := extractHost(baseURL)
	if host == "" {
		return true // can't validate, skip
	}

	conn, err := net.DialTimeout("tcp", host+":443", 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func extractHost(rawURL string) string {
	// Strip protocol and path
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	if idx := strings.Index(rawURL, "/"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.Index(rawURL, ":"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	return rawURL
}

// ── Test Output ──

func printTestResults(rows *db.Rows) {
	defer rows.Close()

	cols := rows.Columns()
	count := 0
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			case nil:
				row[i] = "NULL"
			default:
				row[i] = fmt.Sprint(val)
			}
		}
		if count == 0 {
			fmt.Printf("  %sSQL: SELECT ... LIMIT 10%s\n", "\033[2m", "\033[0m")
			fmt.Println()
			header := "  "
			for _, c := range cols {
				header += fmt.Sprintf("│ %-20s ", c)
			}
			fmt.Println(header)
			fmt.Println("  " + strings.Repeat("─", len(cols)*23+1))
		}
		line := "  "
		for _, v := range row {
			if len(v) > 18 {
				v = v[:18] + ".."
			}
			line += fmt.Sprintf("│ %-20s ", v)
		}
		fmt.Println(line)
		count++
	}

	if count == 0 {
		fmt.Printf("  %s✓ Connected — no tables found (empty database)%s\n", "\033[32m", "\033[0m")
	} else {
		fmt.Printf("\n  %s✓%s %s%d tables found%s\n", "\033[32m", "\033[0m", "\033[2m", count, "\033[0m")
	}
}

// ── Database Selection Wizard ──

func selectDatabase(reader *bufio.Reader, green, yellow, dim, bold, reset string) (db.Database, string, string) {
	// Auto-detect running databases
	fmt.Printf("  %sScanning for databases...%s\n", dim, reset)
	detected := detectDatabases()

	if len(detected) > 0 {
		fmt.Println()
		for i, d := range detected {
			fmt.Printf("  %d) %s%s%s\n", i+1, green, d.label, reset)
		}
		fmt.Println()
		fmt.Printf("  %sUse detected database [1], or:%s\n", bold, reset)
		fmt.Printf("  %sm   Enter a connection string manually%s\n", dim, reset)
		fmt.Printf("  %sd   Start with demo data (no real DB needed)%s\n", dim, reset)
		fmt.Printf("  %ss   Skip database (configure later)%s\n", dim, reset)
		fmt.Print("\n  → ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "" || choice == "1" {
			d := detected[0]
			conn, err := db.Connect(d.dsn)
			if err != nil {
				fmt.Printf("  %s⚠ Could not connect: %s%s\n", yellow, friendlyDBError(err), reset)
				fmt.Printf("  %s  Let's try another way...%s\n", dim, reset)
				return askManualOrDemo(reader, green, yellow, dim, bold, reset)
			}
			return conn, conn.Name(), d.dsn
		}
		if choice == "d" {
			return setupDemoDB(dim, yellow, reset)
		}
		if choice == "s" {
			fmt.Printf("  %s⏭ Skipping database for now%s\n", dim, reset)
			return nil, "", ""
		}
		return askManualOrDemo(reader, green, yellow, dim, bold, reset)
	}

	fmt.Printf("  %sNo databases found on localhost.%s\n", dim, reset)
	return askManualOrDemo(reader, green, yellow, dim, bold, reset)
}

func askManualOrDemo(reader *bufio.Reader, green, yellow, dim, bold, reset string) (db.Database, string, string) {
	fmt.Println()
	fmt.Printf("  %sConnect a database:%s\n", bold, reset)
	fmt.Printf("  %sd   Start with demo data (recommended)%s\n", dim, reset)
	fmt.Printf("  %sm   Enter a connection string manually%s\n", dim, reset)
	fmt.Printf("  %ss   Skip (configure later)%s\n", dim, reset)
	fmt.Print("\n  → ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch {
	case choice == "" || choice == "d":
		return setupDemoDB(dim, yellow, reset)
	case choice == "m":
		fmt.Print("\n  Enter DSN (e.g. postgres://user:pass@localhost/mydb):\n  → ")
		manualDSN, _ := reader.ReadString('\n')
		manualDSN = strings.TrimSpace(manualDSN)
		if manualDSN != "" {
			conn, err := db.Connect(manualDSN)
			if err != nil {
				fmt.Printf("  %s⚠ %s%s\n", yellow, friendlyDBError(err), reset)
				fmt.Printf("  %s  Tip: Check host, port, and credentials%s\n", dim, reset)
				return nil, "", ""
			}
			return conn, conn.Name(), manualDSN
		}
		return nil, "", ""
	default:
		fmt.Printf("  %s⏭ Skipping database for now%s\n", dim, reset)
		return nil, "", ""
	}
}

func setupDemoDB(dim, yellow, reset string) (db.Database, string, string) {
	fmt.Printf("  %sSetting up demo database...%s\n", dim, reset)
	conn, name, demoDSN, err := createDemoDB()
	if err != nil {
		fmt.Printf("  %s⚠ Demo setup failed: %s%s\n", yellow, err, reset)
		return nil, "", ""
	}
	return conn, name, demoDSN
}

// ── Error Helpers ──

func friendlyDBError(err error) string {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "connection refused"):
		return "Could not connect: Is the database running?"
	case strings.Contains(msg, "password authentication failed"):
		return "Authentication failed: Check your username and password"
	case strings.Contains(msg, "does not exist"):
		return "Database not found: Check the database name"
	case strings.Contains(msg, "dial tcp"):
		return "Could not reach host: Check the hostname and port"
	case strings.Contains(msg, "i/o timeout"):
		return "Connection timed out: Is the database reachable?"
	default:
		return msg
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
