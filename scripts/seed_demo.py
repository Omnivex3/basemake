#!/usr/bin/env python3
"""Seed ghost_demo and mastodon_demo with realistic data using Faker."""
import subprocess, psycopg2, random
from datetime import datetime, timedelta, timezone

try:
    from faker import Faker
except ImportError:
    subprocess.run(["pip", "install", "faker", "psycopg2-binary"], check=True)
    from faker import Faker

fake = Faker()
Faker.seed(42)
random.seed(42)
now = datetime.now(timezone.utc)

DSN = "host=localhost port=5433 dbname={} user=postgres password=postgres"

def get_conn(dbname):
    return psycopg2.connect(DSN.format(dbname))

def seed_ghost():
    conn = get_conn("ghost_demo")
    cur = conn.cursor()

    # Schema
    cur.execute("""
        CREATE TABLE IF NOT EXISTS authors (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            slug VARCHAR(255) UNIQUE NOT NULL,
            bio TEXT,
            created_at TIMESTAMPTZ DEFAULT NOW()
        );
        CREATE TABLE IF NOT EXISTS tags (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100) NOT NULL UNIQUE,
            slug VARCHAR(100) UNIQUE NOT NULL,
            description TEXT
        );
        CREATE TABLE IF NOT EXISTS posts (
            id SERIAL PRIMARY KEY,
            title VARCHAR(500) NOT NULL,
            slug VARCHAR(500) UNIQUE NOT NULL,
            excerpt TEXT,
            content TEXT,
            status VARCHAR(20) DEFAULT 'draft',
            published_at TIMESTAMPTZ,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW(),
            author_id INTEGER REFERENCES authors(id)
        );
        CREATE TABLE IF NOT EXISTS posts_tags (
            post_id INTEGER REFERENCES posts(id),
            tag_id INTEGER REFERENCES tags(id),
            PRIMARY KEY (post_id, tag_id)
        );
    """)
    conn.commit()

    # Wipe & reseed
    cur.execute("TRUNCATE posts_tags, posts, tags, authors RESTART IDENTITY CASCADE")

    # Authors (25)
    authors = []
    for _ in range(25):
        name = fake.unique.name()
        slug = name.lower().replace(" ", "-").replace(".", "").replace("'", "")
        authors.append((
            name,
            fake.unique.email(),
            slug,
            fake.text(max_nb_chars=120),
            now - timedelta(days=random.randint(30, 365))
        ))
    cur.executemany(
        "INSERT INTO authors (name, email, slug, bio, created_at) VALUES (%s,%s,%s,%s,%s)",
        authors
    )
    conn.commit()

    # Tags (12)
    tags = ["Tech", "Design", "Product", "Engineering", "AI", "DevOps",
            "Open Source", "Tutorial", "News", "Opinion", "Career", "Data"]
    tag_data = [(t, t.lower(), fake.text(max_nb_chars=60)) for t in tags]
    cur.executemany(
        "INSERT INTO tags (name, slug, description) VALUES (%s,%s,%s)",
        tag_data
    )
    conn.commit()

    # Posts (200)
    posts = []
    for _ in range(200):
        title = fake.sentence(nb_words=4, variable_nb_words=True).rstrip(".")
        slug = title.lower().replace(" ", "-").replace(".", "").replace("'", "")[:200]
        status = random.choices(["published", "draft", "published", "published"], weights=[1,1,2,2])[0]
        published_at = None
        if status == "published":
            published_at = now - timedelta(days=random.randint(0, 90), hours=random.randint(0, 23))
        posts.append((
            title,
            slug,
            fake.text(max_nb_chars=200),
            fake.text(max_nb_chars=2000),
            status,
            published_at,
            now - timedelta(days=random.randint(0, 180)),
            random.randint(1, 25)
        ))
    cur.executemany(
        "INSERT INTO posts (title, slug, excerpt, content, status, published_at, created_at, author_id) VALUES (%s,%s,%s,%s,%s,%s,%s,%s)",
        posts
    )
    conn.commit()

    # Posts-Tags (avg 2 tags per post)
    pt = []
    for pid in range(1, 201):
        for tid in random.sample(range(1, 13), random.randint(1, 3)):
            pt.append((pid, tid))
    cur.executemany(
        "INSERT INTO posts_tags (post_id, tag_id) VALUES (%s,%s) ON CONFLICT DO NOTHING",
        pt
    )
    conn.commit()

    cur.execute("SELECT count(*) FROM authors")
    a = cur.fetchone()[0]
    cur.execute("SELECT count(*) FROM posts")
    p = cur.fetchone()[0]
    cur.execute("SELECT count(*) FROM tags")
    t = cur.fetchone()[0]
    cur.close(); conn.close()
    print(f"✓ Ghost: {a} authors, {p} posts, {t} tags")

def seed_mastodon():
    conn = get_conn("mastodon_demo")
    cur = conn.cursor()

    # Schema
    cur.execute("""
        CREATE TABLE IF NOT EXISTS accounts (
            id SERIAL PRIMARY KEY,
            username VARCHAR(100) UNIQUE NOT NULL,
            display_name VARCHAR(200),
            bio TEXT,
            followers_count INTEGER DEFAULT 0,
            following_count INTEGER DEFAULT 0,
            created_at TIMESTAMPTZ DEFAULT NOW()
        );
        CREATE TABLE IF NOT EXISTS statuses (
            id SERIAL PRIMARY KEY,
            account_id INTEGER REFERENCES accounts(id),
            content TEXT,
            visibility VARCHAR(20) DEFAULT 'public',
            replies_count INTEGER DEFAULT 0,
            reblogs_count INTEGER DEFAULT 0,
            favourites_count INTEGER DEFAULT 0,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
        CREATE TABLE IF NOT EXISTS follows (
            id SERIAL PRIMARY KEY,
            account_id INTEGER REFERENCES accounts(id),
            target_account_id INTEGER REFERENCES accounts(id),
            created_at TIMESTAMPTZ DEFAULT NOW(),
            UNIQUE(account_id, target_account_id)
        );
        CREATE TABLE IF NOT EXISTS status_tags (
            status_id INTEGER REFERENCES statuses(id),
            tag VARCHAR(100) NOT NULL,
            PRIMARY KEY (status_id, tag)
        );
    """)
    conn.commit()

    # Wipe & reseed
    cur.execute("TRUNCATE status_tags, follows, statuses, accounts RESTART IDENTITY CASCADE")

    # Accounts (30)
    accounts = []
    for _ in range(30):
        username = fake.unique.user_name()[:50]
        accounts.append((
            username,
            fake.name(),
            fake.text(max_nb_chars=150),
            random.randint(0, 5000),
            random.randint(0, 500),
            now - timedelta(days=random.randint(1, 365))
        ))
    cur.executemany(
        "INSERT INTO accounts (username, display_name, bio, followers_count, following_count, created_at) VALUES (%s,%s,%s,%s,%s,%s)",
        accounts
    )
    conn.commit()

    # Statuses (500)
    statuses = []
    for _ in range(500):
        statuses.append((
            random.randint(1, 30),
            fake.text(max_nb_chars=500),
            random.choices(["public", "unlisted", "private"], weights=[7,2,1])[0],
            random.randint(0, 20),
            random.randint(0, 50),
            random.randint(0, 100),
            now - timedelta(days=random.randint(0, 60), hours=random.randint(0, 23))
        ))
    cur.executemany(
        "INSERT INTO statuses (account_id, content, visibility, replies_count, reblogs_count, favourites_count, created_at) VALUES (%s,%s,%s,%s,%s,%s,%s)",
        statuses
    )
    conn.commit()

    # Follows (200)
    follows = set()
    while len(follows) < 200:
        a = random.randint(1, 30)
        b = random.randint(1, 30)
        if a != b:
            follows.add((a, b))
    cur.executemany(
        "INSERT INTO follows (account_id, target_account_id) VALUES (%s,%s) ON CONFLICT DO NOTHING",
        list(follows)
    )
    conn.commit()

    # Status tags
    hashtags = ["tech", "devops", "ai", "python", "golang", "cloud",
                "opensource", "kubernetes", "database", "security",
                "design", "productivity", "remote", "career", "climate"]
    st = []
    for sid in range(1, 501):
        for tag in random.sample(hashtags, random.randint(0, 3)):
            st.append((sid, tag))
    cur.executemany(
        "INSERT INTO status_tags (status_id, tag) VALUES (%s,%s) ON CONFLICT DO NOTHING",
        st
    )
    conn.commit()

    cur.execute("SELECT count(*) FROM accounts")
    a = cur.fetchone()[0]
    cur.execute("SELECT count(*) FROM statuses")
    s = cur.fetchone()[0]
    cur.execute("SELECT count(*) FROM follows")
    f = cur.fetchone()[0]
    cur.close(); conn.close()
    print(f"✓ Mastodon: {a} accounts, {s} statuses, {f} follows")

if __name__ == "__main__":
    seed_ghost()
    seed_mastodon()
    print("Done.")
