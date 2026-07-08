#!/usr/bin/env python3
"""Seed a Discourse-style Postgres database with realistic demo data."""
import subprocess, sys, random, os
from datetime import datetime, timedelta, timezone

try:
    import psycopg2
except ImportError:
    subprocess.run([sys.executable, "-m", "pip", "install", "psycopg2-binary", "faker"], check=True)
    import psycopg2
    from faker import Faker

from faker import Faker

fake = Faker()
Faker.seed(42)
random.seed(42)
now = datetime.now(timezone.utc)

# Config
DSN = os.environ.get("DISCOURSE_DSN", "host=localhost port=5434 dbname=discourse user=postgres password=postgres")
USER_COUNT = 100
TOPIC_COUNT = 1500
POST_COUNT = 5000
TAG_COUNT = 20
CATEGORIES = [
    ("General", "general", "General discussion about everything"),
    ("Feedback", "feedback", "Share feedback about the product"),
    ("Support", "support", "Get help and support"),
    ("Dev", "dev", "Software development discussions"),
    ("Design", "design", "UI, UX, and design topics"),
    ("Community", "community", "Community events and announcements"),
    ("Engineering", "engineering", "Engineering deep-dives and architecture"),
    ("Product", "product", "Product updates and roadmap"),
]

def get_conn():
    return psycopg2.connect(DSN)

def seed():
    conn = get_conn()
    cur = conn.cursor()

    print("Creating Discourse schema...")

    cur.execute("""
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            username VARCHAR(100) UNIQUE NOT NULL,
            name VARCHAR(200),
            email VARCHAR(255) UNIQUE NOT NULL,
            admin BOOLEAN DEFAULT FALSE,
            moderator BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            last_seen_at TIMESTAMPTZ DEFAULT NOW(),
            trust_level INTEGER DEFAULT 0
        );

        CREATE TABLE IF NOT EXISTS categories (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100) NOT NULL,
            slug VARCHAR(100) UNIQUE NOT NULL,
            description TEXT,
            color VARCHAR(6) DEFAULT 'ffffff',
            position INTEGER DEFAULT 0,
            parent_category_id INTEGER REFERENCES categories(id),
            topics_count INTEGER DEFAULT 0,
            created_at TIMESTAMPTZ DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS topics (
            id SERIAL PRIMARY KEY,
            title VARCHAR(500) NOT NULL,
            slug VARCHAR(500) UNIQUE NOT NULL,
            user_id INTEGER REFERENCES users(id),
            category_id INTEGER REFERENCES categories(id),
            created_at TIMESTAMPTZ DEFAULT NOW(),
            bumped_at TIMESTAMPTZ DEFAULT NOW(),
            last_posted_at TIMESTAMPTZ,
            views INTEGER DEFAULT 0,
            posts_count INTEGER DEFAULT 0,
            like_count INTEGER DEFAULT 0,
            archived BOOLEAN DEFAULT FALSE,
            pinned BOOLEAN DEFAULT FALSE,
            closed BOOLEAN DEFAULT FALSE,
            visible BOOLEAN DEFAULT TRUE
        );

        CREATE TABLE IF NOT EXISTS posts (
            id SERIAL PRIMARY KEY,
            topic_id INTEGER REFERENCES topics(id),
            user_id INTEGER REFERENCES users(id),
            raw TEXT NOT NULL,
            cooked TEXT,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW(),
            reply_to_post_number INTEGER,
            like_count INTEGER DEFAULT 0,
            sort_order INTEGER DEFAULT 0
        );

        CREATE TABLE IF NOT EXISTS tags (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100) UNIQUE NOT NULL,
            description TEXT,
            topics_count INTEGER DEFAULT 0,
            created_at TIMESTAMPTZ DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS topic_tags (
            topic_id INTEGER REFERENCES topics(id),
            tag_id INTEGER REFERENCES tags(id),
            PRIMARY KEY (topic_id, tag_id)
        );

        CREATE INDEX IF NOT EXISTS idx_topics_category_id ON topics(category_id);
        CREATE INDEX IF NOT EXISTS idx_topics_user_id ON topics(user_id);
        CREATE INDEX IF NOT EXISTS idx_topics_created_at ON topics(created_at);
        CREATE INDEX IF NOT EXISTS idx_topics_bumped_at ON topics(bumped_at);
        CREATE INDEX IF NOT EXISTS idx_posts_topic_id ON posts(topic_id);
        CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
        CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at);
        CREATE INDEX IF NOT EXISTS idx_topic_tags_tag_id ON topic_tags(tag_id);
    """)
    conn.commit()

    # Clear existing data
    cur.execute("TRUNCATE topic_tags, posts, topics, tags, categories, users RESTART IDENTITY CASCADE")
    conn.commit()

    # ── Users ──
    print(f"Creating {USER_COUNT} users...")
    users = []
    for _ in range(USER_COUNT):
        name = fake.name()
        username = fake.unique.user_name()[:50]
        email = fake.unique.email()
        admin = random.random() < 0.05
        moderator = random.random() < 0.1
        tl = random.choices([0, 1, 2, 3, 4], weights=[20, 30, 25, 15, 10])[0]
        last_seen = now - timedelta(days=random.randint(0, 60), hours=random.randint(0, 23))
        users.append((username, name, email, admin, moderator, now - timedelta(days=random.randint(30, 365)), last_seen, tl))
    cur.executemany(
        "INSERT INTO users (username, name, email, admin, moderator, created_at, last_seen_at, trust_level) VALUES (%s,%s,%s,%s,%s,%s,%s,%s)",
        users
    )
    conn.commit()

    # ── Categories ──
    print(f"Creating {len(CATEGORIES)} categories...")
    colors = ["e45735", "f1a832", "2596be", "8b6fc7", "72b84b", "e68442", "4a9b8f", "c7526a"]
    cats_created = []
    for i, (name, slug, desc) in enumerate(CATEGORIES):
        cur.execute(
            "INSERT INTO categories (name, slug, description, color, position, created_at) VALUES (%s,%s,%s,%s,%s,%s) RETURNING id",
            (name, slug, desc, colors[i % len(colors)], i, now - timedelta(days=365))
        )
        cats_created.append(cur.fetchone()[0])
    conn.commit()

    # ── Tags ──
    print(f"Creating {TAG_COUNT} tags...")
    tag_names = [
        "discussion", "help", "bug", "feature-request", "tutorial",
        "performance", "security", "announcement", "question", "show-and-tell",
        "database", "deployment", "docker", "api", "backend",
        "frontend", "testing", "architecture", "bash", "monitoring"
    ]
    for tn in tag_names[:TAG_COUNT]:
        cur.execute(
            "INSERT INTO tags (name, description, created_at) VALUES (%s,%s,%s)",
            (tn, fake.text(max_nb_chars=100), now - timedelta(days=365))
        )
    conn.commit()

    # ── Topics ──
    print(f"Creating {TOPIC_COUNT} topics...")
    topics_batch = []
    for _ in range(TOPIC_COUNT):
        title = fake.sentence(nb_words=5, variable_nb_words=True).rstrip(".")[:200]
        slug = title.lower().replace(" ", "-").replace(".", "").replace("'", "")[:200] + f"-{random.randint(100,999)}"
        uid = random.randint(1, USER_COUNT)
        cid = random.choice(cats_created)
        created = now - timedelta(days=random.randint(0, 180), hours=random.randint(0, 23))
        bumped = created + timedelta(hours=random.randint(0, 48))
        last_posted = bumped if random.random() > 0.3 else None
        views = random.randint(10, 5000)
        likes = random.randint(0, 100)
        archived = random.random() < 0.02
        pinned = random.random() < 0.03
        closed = random.random() < 0.05
        topics_batch.append((title, slug, uid, cid, created, bumped, last_posted, views, likes, archived, pinned, closed))
    cur.executemany(
        "INSERT INTO topics (title, slug, user_id, category_id, created_at, bumped_at, last_posted_at, views, like_count, archived, pinned, closed) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)",
        topics_batch
    )
    conn.commit()

    # ── Posts ──
    print(f"Creating ~{POST_COUNT} posts...")
    posts_batch = []
    post_num = {}  # topic_id → count
    for _ in range(POST_COUNT):
        tid = random.randint(1, TOPIC_COUNT)
        uid = random.randint(1, USER_COUNT)
        post_num[tid] = post_num.get(tid, 0) + 1
        so = post_num[tid]
        reply_to = None
        if so > 1 and random.random() < 0.6:
            reply_to = random.randint(1, so - 1)
        created = now - timedelta(days=random.randint(0, 180), hours=random.randint(0, 23))
        likes = random.randint(0, 30)
        raw = fake.paragraph(nb_sentences=random.randint(3, 12))
        posts_batch.append((tid, uid, raw, None, created, created, reply_to, likes, so))
    cur.executemany(
        "INSERT INTO posts (topic_id, user_id, raw, cooked, created_at, updated_at, reply_to_post_number, like_count, sort_order) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)",
        posts_batch
    )
    conn.commit()

    # ── Topic-Tags ──
    print("Creating topic-tag mappings...")
    tt_batch = set()
    for tid in range(1, TOPIC_COUNT + 1):
        for _ in range(random.randint(0, 3)):
            tag_id = random.randint(1, TAG_COUNT)
            tt_batch.add((tid, tag_id))
    cur.executemany(
        "INSERT INTO topic_tags (topic_id, tag_id) VALUES (%s,%s) ON CONFLICT DO NOTHING",
        list(tt_batch)
    )
    conn.commit()

    # ── Update counters ──
    cur.execute("""
        UPDATE categories c SET topics_count = (
            SELECT count(*) FROM topics t WHERE t.category_id = c.id
        )
    """)
    cur.execute("""
        UPDATE tags t SET topics_count = (
            SELECT count(*) FROM topic_tags tt WHERE tt.tag_id = t.id
        )
    """)
    cur.execute("""
        UPDATE topics t SET posts_count = (
            SELECT count(*) FROM posts p WHERE p.topic_id = t.id
        )
    """)
    conn.commit()

    # Verify
    for tbl in ["users", "categories", "topics", "posts", "tags", "topic_tags"]:
        cur.execute(f"SELECT count(*) FROM {tbl}")
        print(f"  ✓ {tbl}: {cur.fetchone()[0]}")

    cur.close()
    conn.close()
    print("\nDiscourse demo database ready.")

if __name__ == "__main__":
    seed()
