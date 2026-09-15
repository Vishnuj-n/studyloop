#!/usr/bin/env python3
"""
StudyLoop Database Diagnostic Tool

NOTE ON DATABASE TARGETS:
- By default (no flags), this script inspects the production desktop app database located at:
    %APPDATA%\\Studyloop\\Studyloop.db (Windows: C:\\Users\\<user>\\AppData\\Roaming\\Studyloop\\Studyloop.db)
- Use `--dev` to inspect the local development database located at:
    dev_data/Studyloop.db
- Use `--db <path>` to specify a custom database file path.

Usage:
    python scripts/inspect_db.py              # Inspect installed / AppData database
    python scripts/inspect_db.py --dev        # Inspect dev_data/Studyloop.db
    python scripts/inspect_db.py --db path.db # Inspect specific database file

Checks performed:
- Active Profile & User Settings
- Active Notebooks & Topic Reading Progress
- Study Queue Tasks (Active / Pending / Completed)
- FSRS Due Flashcards vs Linked Review Tasks
- Invariant & Queue Deadlock Detection
"""

import argparse
import os
import sqlite3
import sys
import time

def get_db_path(dev=False, custom_path=None):
    if custom_path:
        return os.path.abspath(custom_path)
    if dev:
        return os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "dev_data", "Studyloop.db"))
    # Default: production / installed AppData location
    appdata = os.getenv("APPDATA")
    if not appdata:
        appdata = os.path.expanduser("~\\AppData\\Roaming")
    return os.path.join(appdata, "Studyloop", "Studyloop.db")

def inspect_db(db_path):
    if not os.path.exists(db_path):
        print(f"[ERROR] Database file not found at: {db_path}", file=sys.stderr)
        sys.exit(1)

    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    c = conn.cursor()

    print("=== StudyLoop Database Diagnostics ===")
    print(f"Path: {db_path}\n")

    # 1. Profile & Settings
    print("--- 1. ACTIVE PROFILE & USER SETTINGS ---")
    active_profile_id = ""
    settings_row = c.execute("SELECT * FROM user_settings LIMIT 1").fetchone()
    if settings_row:
        s = dict(settings_row)
        active_profile_id = s.get("active_profile_id", "")
        print(f"Active Profile ID: {active_profile_id}")
        print(f"Target Session Words: {s.get('target_session_words')}")
        print(f"Min Session Words: {s.get('min_session_words')}")
        print(f"Max Flashcards/Session: {s.get('max_flashcards_per_session')}")
        print(f"Skip to Reading Active: {bool(s.get('skip_to_reading_active'))}")
    else:
        print("[WARN] No user settings row found!")

    profiles = c.execute("SELECT id, name, deadline_at FROM study_profiles").fetchall()
    print("\nProfiles:")
    for p in profiles:
        active_mark = " (ACTIVE)" if p["id"] == active_profile_id else ""
        print(f"  - [{p['id']}]: {p['name']}{active_mark}")

    # 2. Active Notebooks
    print("\n--- 2. NOTEBOOKS ---")
    notebooks = c.execute("SELECT id, title, study_status, status, profile_id FROM notebooks").fetchall()
    for nb in notebooks:
        print(f"  - [{nb['id']}] {nb['title']}")
        print(f"    study_status: {nb['study_status']} | status: {nb['status']} | profile_id: {nb['profile_id']}")

    # 3. Queue Tasks
    print("\n--- 3. STUDY QUEUE (ACTIVE & PENDING) ---")
    queue_tasks = c.execute(
        "SELECT id, notebook_id, topic_id, task_type, status, priority, start_page, end_page, created_at "
        "FROM study_queue WHERE status IN ('ACTIVE', 'PENDING') ORDER BY priority DESC, created_at ASC"
    ).fetchall()

    if not queue_tasks:
        print("  [ALERT] Queue is completely EMPTY (0 active or pending tasks)!")
    else:
        for t in queue_tasks:
            print(f"  - [{t['status']}] Type: {t['task_type']} | Task ID: {t['id']}")
            print(f"    Notebook: {t['notebook_id']} | Topic: {t['topic_id']} | Pages: {t['start_page']}-{t['end_page']}")

    # 4. Due Flashcards
    print("\n--- 4. FLASHCARD STATS ---")
    total_cards = c.execute("SELECT count(*) FROM fsrs_cards").fetchone()[0]
    now_ts = int(time.time())
    due_cards = c.execute("SELECT count(*) FROM fsrs_cards WHERE suspended = 0 AND due_at <= ?", (now_ts,)).fetchone()[0]
    print(f"  Total Cards: {total_cards} | Due Right Now: {due_cards}")

    # 5. Diagnostic Summary
    print("\n--- 5. DIAGNOSIS ---")
    reading_tasks = [t for t in queue_tasks if t['task_type'] in ('READING', 'REREAD')]
    review_tasks = [t for t in queue_tasks if t['task_type'] == 'FLASHCARD_REVIEW']
    quiz_tasks = [t for t in queue_tasks if t['task_type'] in ('QUIZ', 'MILESTONE_EXAM')]

    # Check how many cards are linked to review tasks
    linked_cards = c.execute(
        "SELECT count(DISTINCT card_id) FROM review_task_cards rtc "
        "JOIN study_queue sq ON sq.id = rtc.task_id "
        "WHERE sq.status IN ('PENDING', 'ACTIVE')"
    ).fetchone()[0]

    unlinked_due = due_cards - linked_cards
    if unlinked_due < 0:
        unlinked_due = 0

    print(f"  - Reading/Reread Tasks in Queue: {len(reading_tasks)}")
    print(f"  - Quiz/Exam Tasks in Queue: {len(quiz_tasks)}")
    print(f"  - Review Tasks in Queue: {len(review_tasks)}")
    print(f"  - Cards linked to pending review tasks: {linked_cards}")
    print(f"  - Unlinked due cards (QueryDueReviewCards): {unlinked_due}")

    if not reading_tasks and not quiz_tasks:
        if review_tasks:
            print("\n  [!] QUEUE DEADLOCK DETECTED:")
            print("  1. Each active notebook has a PENDING FLASHCARD_REVIEW task in `study_queue`.")
            print("  2. `EnsurePendingReadingTasksForActiveNotebooks` checks:")
            print("     `NOT EXISTS (SELECT 1 FROM study_queue sq WHERE sq.notebook_id = n.id AND sq.status IN ('PENDING', 'ACTIVE'))`")
            print("     Because the review tasks are in PENDING status, the query believes the notebook already has an active task and skips creating READING tasks.")
            print("  3. Meanwhile, all due cards are tied up in `review_task_cards`, so `QueryDueReviewCards` returns 0 unlinked cards.")
            print("  4. The frontend Dashboard filters out all review tasks into `reviewTask` and sees `dueReviewCards == 0`.")
            print("     -> `isReviewHero` is false, `focusHeroTask` is null, and `queueTasks` is empty.")
            print("  5. Result: The dashboard shows a completely blank 'Today's Tasks' page.")
        else:
            print("\n  [!] Queue is empty and no reading tasks were generated.")
    else:
        print("\n  [✓] Queue state healthy.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Inspect StudyLoop SQLite database state (default: %APPDATA%\\Studyloop\\Studyloop.db)")
    parser.add_argument("--dev", action="store_true", help="Inspect local development database at dev_data/Studyloop.db")
    parser.add_argument("--db", type=str, default=None, help="Custom path to Studyloop.db file")
    args = parser.parse_args()
    inspect_db(get_db_path(dev=args.dev, custom_path=args.db))
