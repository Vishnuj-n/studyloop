# Cloud Dashboard & Cloud Server Developer Handover Guide

Welcome! This guide is designed to help a developer take over, configure, debug, and enhance the **AI Tutor Cloud Dashboard** (`cloud-dashboard/`) and **Go Cloud Server** (`cmd/cloud-server/`).

---

## 1. System Architecture Overview

The Cloud Architecture connects student desktop client instances with teacher monitoring web applications.

```
+------------------------+             +--------------------------+
|  Student Desktop App   |             |   Teacher Cloud Portal   |
| (Wails / Go + SQLite)  |             |      (Vue 3 + Vite)      |
+-----------+------------+             +------------+-------------+
            |                                       |
            | 1. Delta Sync (POST)                  | 2. Direct REST / RPC
            v                                       v
+-----------------------------------------------------------------+
|                      Supabase (PostgreSQL)                      |
|   - Tables: user_accounts, student_notebooks, student_review_logs|
|   - RPCs: login_user, signup_user, handle_cloud_sync, dashboard |
|   - Storage: 'assignments' PDF bucket                           |
+-----------------------------------------------------------------+
                                ^
                                | 3. Optional Go Cloud Proxy Server
                                |    (cmd/cloud-server)
```

---

## 2. Setting Up Backend Access

You have two ways to give another developer access to the Supabase backend:

---

### Option A: Invite Developer Directly to Existing Supabase Project (Recommended)
If you want your friend to work on your **existing database** with all current data, user accounts, and storage buckets:

1. Open your project on [supabase.com](https://supabase.com).
2. Go to **Organization Settings** -> **Members** (or **Project Settings** -> **Team**).
3. Click **Invite Member**, enter your friend's email address, and select a role (e.g., `Developer` or `Admin`).
4. Once they accept the invitation email, they will have full access to the project SQL editor, tables, and settings.
5. Provide them with the project's **Supabase URL** and **Anon / Service Role Key** (found in `Project Settings` -> `API`) so they can configure their local `.env` files.

---

### Option B: Provision a Fresh Supabase Instance
If the developer prefers to set up an isolated development database from scratch:

1. Create a new project at [supabase.com](https://supabase.com).
2. Go to the **SQL Editor** in the new Supabase project dashboard.
3. Open [`cloud-dashboard/supabase/setup_all.sql`](../cloud-dashboard/supabase/setup_all.sql).
4. Copy and paste the entire script into the SQL Editor and click **Run**.
5. Save your **Project URL** and **Anon Key** from `Project Settings -> API`.


---

## 3. Environment Variables Configuration

Copy the example environment files to create local `.env` files:

### Cloud Dashboard (`cloud-dashboard/.env`)
```bash
cp cloud-dashboard/.env.example cloud-dashboard/.env
```
Update `cloud-dashboard/.env`:
```env
VITE_SUPABASE_URL=https://<your-project-ref>.supabase.co
VITE_SUPABASE_ANON_KEY=<your-anon-key>
VITE_CLASSROOM_CODE=BIO101
```

### Go Cloud Server (`cmd/cloud-server/.env`)
```bash
cp cmd/cloud-server/.env.example cmd/cloud-server/.env
```
Update `cmd/cloud-server/.env`:
```env
SUPABASE_URL=https://<your-project-ref>.supabase.co
SUPABASE_SERVICE_ROLE_KEY=<your-service-role-key>
PORT=8080
CORS_ALLOWED_ORIGINS=http://localhost:5173
```

---

## 4. Running & Debugging the Components

### A. Running the Cloud Server (`cmd/cloud-server`)
```bash
# From project root:
go run ./cmd/cloud-server
```
Health Check:
```bash
curl http://localhost:8080/health
```

### B. Running the Cloud Dashboard (`cloud-dashboard`)
```bash
cd cloud-dashboard
npm install
npm run dev
```
Open `http://localhost:5173` in your browser.

---

## 5. Developer Takeover Checklist & Issue Roadmap

Here is the current status and roadmap of tasks for fixing and upgrading the Cloud Dashboard & Server:

### 1. Verification of Auth & CORS Flow
- **Goal:** Ensure cross-origin requests from `localhost:5173` to `localhost:8080` pass `x-session-token` headers correctly.
- **Verification:** Login via Cloud Dashboard web app and check browser network console for CORS preflight (`OPTIONS`) handling.

### 2. Assignment PDF Storage Bucket Uploads
- **Goal:** Teachers can upload PDFs to the Supabase Storage bucket `assignments`.
- **Key Code:** [`cloud-dashboard/src/composables/useDashboard.js`](../cloud-dashboard/src/composables/useDashboard.js)
- **Check:** Verify RLS policy on bucket `assignments` allows inserts using session tokens.

### 3. Student Socratic Red Alert Filter
- **Goal:** Highlighting students who are stuck in consecutive quiz failure loops (`external_help_required = true`).
- **Check:** Verify that `student_notebooks` rows update `external_help_required = true` during `handle_cloud_sync` RPC calls.

---

## 6. Building for Production

When bundling the Desktop client for distribution:
```bash
# Pass environment variables for production sync endpoints:
$env:CLOUD_SYNC_URL = "https://<your-project-ref>.supabase.co/rest/v1/rpc/handle_cloud_sync"
$env:SUPABASE_ANON_KEY = "<your-anon-key>"

python scripts/build.py
```
