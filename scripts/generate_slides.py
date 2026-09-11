import base64
import os
import subprocess
import sys
import time
import urllib.request
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
FRONTEND_DIR = ROOT_DIR / "frontend"
OUTPUT_FILE = ROOT_DIR / "slides.html"
PORT = 4173
BASE_URL = f"http://localhost:{PORT}"

ROUTES = [
    {"name": "Dashboard", "path": "/#/dashboard", "desc": "Main study overview, stats, and queue"},
    {"name": "Notebooks", "path": "/#/notebooks", "desc": "Document management & study materials"},
    {"name": "Flashcards", "path": "/#/flashcards", "desc": "FSRS Review and flashcard practice"},
    {"name": "Quiz Mode", "path": "/#/quiz", "desc": "Interactive study quizzes & rescue sessions"},
    {"name": "Smart Reader", "path": "/#/reader", "desc": "PDF / Text reader with AI assistance"},
    {"name": "AI Tutor", "path": "/#/tutor", "desc": "Socratic dialogue and study guide chat"},
    {"name": "Written Assessment", "path": "/#/examiner", "desc": "Written mock tests and evaluation"},
    {"name": "Rewards & Streaks", "path": "/#/rewards", "desc": "Study streak tracker & gamification"},
    {"name": "Extensions", "path": "/#/extensions", "desc": "Browser & app integrations"},
    {"name": "Settings", "path": "/#/settings", "desc": "AI Provider & app configuration"},
    {"name": "Onboarding", "path": "/#/onboarding", "desc": "First-time setup experience"},
]

PORTS_TO_CHECK = [34115, 5173, 3000, 4173]

def is_server_running(url):
    try:
        req = urllib.request.urlopen(url, timeout=1)
        return req.status in (200, 304)
    except Exception:
        return False

def find_active_server():
    for port in PORTS_TO_CHECK:
        url = f"http://localhost:{port}"
        if is_server_running(url):
            return url
    return None

def main():
    print("[+] Starting Slide Deck Generation...")

    vite_process = None
    target_url = find_active_server()

    if target_url:
        print(f"[+] Connected to live running server at: {target_url} (Capturing real live data!)")
    else:
        target_url = "http://localhost:4173"
        print("[+] No active dev server found. Building frontend production bundle...")
        subprocess.run("npm run build", cwd=FRONTEND_DIR, shell=True, check=True)

        print("[+] Starting Vite Preview server on port 4173...")
        vite_process = subprocess.Popen("npx vite preview --port 4173", cwd=FRONTEND_DIR, shell=True)

        for _ in range(30):
            time.sleep(0.5)
            if is_server_running(target_url):
                break

    if not is_server_running(target_url):
        print("[!] Could not connect to preview server.")
        if vite_process:
            vite_process.kill()
        sys.exit(1)

    from playwright.sync_api import sync_playwright

    slides_data = []

    print("[+] Capturing routes via Playwright Chromium...")
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(viewport={"width": 1440, "height": 900}, device_scale_factor=2)
        # Enforce dark-gruvbox theme across all pages
        context.add_init_script("document.documentElement.setAttribute('data-theme', 'dark-gruvbox'); localStorage.setItem('app-theme', 'dark-gruvbox');")
        page = context.new_page()

        # Set onboarded state in localStorage
        page.goto(f"{target_url}/#/dashboard", wait_until="networkidle")
        page.evaluate("localStorage.setItem('studyloop_onboarded', 'true'); localStorage.setItem('app-theme', 'dark-gruvbox'); document.documentElement.setAttribute('data-theme', 'dark-gruvbox');")

        for idx, route in enumerate(ROUTES, 1):
            print(f"  [{idx}/{len(ROUTES)}] Capturing {route['name']} ({route['path']})...")
            page.goto(f"{target_url}{route['path']}", wait_until="networkidle")
            page.wait_for_timeout(600)

            screenshot_bytes = page.screenshot(type="png", full_page=True)
            base64_img = f"data:image/png;base64,{base64.b64encode(screenshot_bytes).decode('utf-8')}"

            slides_data.append({
                "id": idx,
                "title": route["name"],
                "path": route["path"],
                "desc": route["desc"],
                "image": base64_img
            })

        browser.close()

    if vite_process:
        vite_process.terminate()

    print("[+] Assembling slides.html...")
    html_content = generate_html_deck(slides_data)

    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        f.write(html_content)

    print(f"[SUCCESS] Slide deck generated at: {OUTPUT_FILE}")

def generate_html_deck(slides):
    slides_js = str(slides)
    cards_html = ""
    for s in slides:
        cards_html += f"""
    <div class="slide {'active' if s['id'] == 1 else ''}" id="slide-{s['id']}">
      <div class="slide-body">
        <img src="{s['image']}" alt="{s['title']}" />
      </div>
      <div class="slide-footer">
        <div><strong>{s['id']}. {s['title']}</strong> — {s['desc']}</div>
        <div>Route: <code>{s['path']}</code></div>
      </div>
    </div>
"""

    return f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Studyloop — UI Presentation Deck</title>
  <style>
    * {{ box-sizing: border-box; margin: 0; padding: 0; }}
    body {{
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background: #0f172a;
      color: #f8fafc;
      height: 100vh;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }}
    header {{
      height: 56px;
      background: #1e293b;
      border-bottom: 1px solid #334155;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 20px;
      z-index: 10;
    }}
    .brand {{
      display: flex;
      align-items: center;
      gap: 12px;
      font-weight: 700;
      font-size: 1.1rem;
      color: #38bdf8;
    }}
    .brand span {{ color: #94a3b8; font-weight: 400; font-size: 0.9rem; }}
    .controls {{
      display: flex;
      align-items: center;
      gap: 12px;
    }}
    button {{
      background: #334155;
      color: #f8fafc;
      border: 1px solid #475569;
      padding: 6px 14px;
      border-radius: 6px;
      cursor: pointer;
      font-weight: 500;
      font-size: 0.85rem;
      transition: all 0.2s ease;
    }}
    button:hover {{ background: #475569; border-color: #64748b; }}
    button:disabled {{ opacity: 0.4; cursor: not-allowed; }}
    .slide-counter {{ font-size: 0.9rem; color: #94a3b8; font-variant-numeric: tabular-nums; }}
    .main-stage {{
      flex: 1;
      position: relative;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
      background: radial-gradient(circle at center, #1e293b 0%, #0f172a 100%);
    }}
    .slide {{
      display: none;
      width: 100%;
      height: 100%;
      max-width: 1440px;
      max-height: 900px;
      background: #090d16;
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255, 255, 255, 0.1);
      flex-direction: column;
    }}
    .slide.active {{ display: flex; }}
    .slide-body {{
      flex: 1;
      position: relative;
      overflow-y: auto;
      background: #020617;
    }}
    .slide-body img {{
      width: 100%;
      height: auto;
      display: block;
    }}
    .slide-footer {{
      height: 48px;
      background: #0f172a;
      border-top: 1px solid #1e293b;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 20px;
      font-size: 0.85rem;
      color: #94a3b8;
    }}
    .slide-footer strong {{ color: #f1f5f9; }}
    .progress-bar {{
      height: 4px;
      background: #1e293b;
      width: 100%;
    }}
    .progress-fill {{
      height: 100%;
      background: #38bdf8;
      width: 0%;
      transition: width 0.3s ease;
    }}
  </style>
</head>
<body>
  <header>
    <div class="brand">
      🎓 Studyloop <span>App UI Presentation Deck</span>
    </div>
    <div class="controls">
      <button onclick="prevSlide()" id="prevBtn">← Previous</button>
      <span class="slide-counter" id="counter">Slide 1 of {len(slides)}</span>
      <button onclick="nextSlide()" id="nextBtn">Next →</button>
      <button onclick="toggleFullscreen()">⛶ Fullscreen</button>
    </div>
  </header>
  <div class="progress-bar"><div class="progress-fill" id="progress"></div></div>
  <div class="main-stage">
    {cards_html}
  </div>

  <script>
    let currentIndex = 0;
    const totalSlides = {len(slides)};

    function updateSlide() {{
      document.querySelectorAll('.slide').forEach((el, idx) => {{
        el.classList.toggle('active', idx === currentIndex);
      }});
      document.getElementById('counter').innerText = `Slide ${{currentIndex + 1}} of ${{totalSlides}}`;
      document.getElementById('prevBtn').disabled = currentIndex === 0;
      document.getElementById('nextBtn').disabled = currentIndex === totalSlides - 1;
      document.getElementById('progress').style.width = `${{((currentIndex + 1) / totalSlides) * 100}}%`;
    }}

    function prevSlide() {{
      if (currentIndex > 0) {{ currentIndex--; updateSlide(); }}
    }}
    function nextSlide() {{
      if (currentIndex < totalSlides - 1) {{ currentIndex++; updateSlide(); }}
    }}
    function toggleFullscreen() {{
      if (!document.fullscreenElement) {{
        document.documentElement.requestFullscreen();
      }} else {{
        document.exitFullscreen();
      }}
    }}

    window.addEventListener('keydown', (e) => {{
      if (e.key === 'ArrowRight' || e.key === ' ') nextSlide();
      if (e.key === 'ArrowLeft') prevSlide();
      if (e.key.toLowerCase() === 'f') toggleFullscreen();
    }});

    updateSlide();
  </script>
</body>
</html>"""

if __name__ == "__main__":
    main()
