import puppeteer from 'puppeteer';
import { spawn } from 'child_process';
import http from 'http';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.resolve(__dirname, '..');
const frontendDir = path.resolve(rootDir, 'frontend');

const ROUTES = [
  { name: 'Dashboard', path: '/#/dashboard', desc: 'Main study overview, stats, and queue' },
  { name: 'Notebooks', path: '/#/notebooks', desc: 'Document management & study materials' },
  { name: 'Flashcards', path: '/#/flashcards', desc: 'FSRS Review and flashcard practice' },
  { name: 'Quiz Mode', path: '/#/quiz', desc: 'Interactive study quizzes & rescue sessions' },
  { name: 'Smart Reader', path: '/#/reader', desc: 'PDF / Text reader with AI assistance' },
  { name: 'AI Tutor', path: '/#/tutor', desc: 'Socratic dialogue and study guide chat' },
  { name: 'Written Assessment', path: '/#/examiner', desc: 'Written mock tests and evaluation' },
  { name: 'Rewards & Streaks', path: '/#/rewards', desc: 'Study streak tracker & gamification' },
  { name: 'Extensions', path: '/#/extensions', desc: 'Browser & app integrations' },
  { name: 'Settings', path: '/#/settings', desc: 'AI Provider & app configuration' },
  { name: 'Onboarding', path: '/#/onboarding', desc: 'First-time setup experience' },
];

function checkServer(url) {
  return new Promise((resolve) => {
    http.get(url, (res) => {
      resolve(res.statusCode === 200 || res.statusCode === 304);
    }).on('error', () => resolve(false));
  });
}

async function main() {
  console.log('🚀 Starting Slide Deck Generation...');

  let viteProcess = null;
  const PORT = 4173;
  const BASE_URL = `http://localhost:${PORT}`;

  // Check if preview/dev server is already running
  let isRunning = await checkServer(BASE_URL);

  if (!isRunning) {
    console.log('📦 Building frontend production bundle...');
    const buildRes = spawn('npm', ['run', 'build'], { cwd: frontendDir, shell: true, stdio: 'inherit' });
    await new Promise((resolve) => buildRes.on('exit', resolve));

    console.log('🌐 Starting Vite Preview server on port ' + PORT + '...');
    viteProcess = spawn('npx', ['vite', 'preview', '--port', String(PORT)], { cwd: frontendDir, shell: true });

    // Wait for server to start
    for (let i = 0; i < 30; i++) {
      await new Promise((r) => setTimeout(r, 500));
      if (await checkServer(BASE_URL)) {
        isRunning = true;
        break;
      }
    }
  }

  if (!isRunning) {
    console.error('❌ Failed to start frontend preview server.');
    if (viteProcess) viteProcess.kill();
    process.exit(1);
  }

  console.log('📸 Capturing routes via Puppeteer...');
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  // Disable onboarding check redirect and enforce theme in localStorage
  await page.evaluateOnNewDocument(() => {
    localStorage.setItem('studyloop_onboarded', 'true');
    localStorage.setItem('app-theme', 'dark-gruvbox');
    document.documentElement.setAttribute('data-theme', 'dark-gruvbox');
  });
  await page.goto(`${BASE_URL}/#/dashboard`, { waitUntil: 'networkidle0' });

  const slidesData = [];

  for (let i = 0; i < ROUTES.length; i++) {
    const route = ROUTES[i];
    console.log(`  [${i + 1}/${ROUTES.length}] Capturing ${route.name} (${route.path})...`);

    await page.goto(`${BASE_URL}${route.path}`, { waitUntil: 'networkidle0' });
    await page.evaluate(() => new Promise((r) => setTimeout(r, 600)));

    const imageBuffer = await page.screenshot({ type: 'png', fullPage: false });
    const base64Image = `data:image/png;base64,${imageBuffer.toString('base64')}`;

    slidesData.push({
      id: i + 1,
      title: route.name,
      path: route.path,
      desc: route.desc,
      image: base64Image
    });
  }

  await browser.close();
  if (viteProcess) viteProcess.kill();

  console.log('📝 Assembling slides.html...');
  const htmlContent = generateSlidesHTML(slidesData);

  const outputPath = path.resolve(rootDir, 'slides.html');
  fs.writeFileSync(outputPath, htmlContent, 'utf-8');
  console.log(`✅ SUCCESS! Slide deck generated at: ${outputPath}`);
  process.exit(0);
}

function generateSlidesHTML(slides) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Studyloop — UI Presentation Deck</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background: #0f172a;
      color: #f8fafc;
      height: 100vh;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }
    header {
      height: 56px;
      background: #1e293b;
      border-bottom: 1px solid #334155;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 20px;
      z-index: 10;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      font-weight: 700;
      font-size: 1.1rem;
      color: #38bdf8;
    }
    .brand span { color: #94a3b8; font-weight: 400; font-size: 0.9rem; }
    .controls {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    button {
      background: #334155;
      color: #f8fafc;
      border: 1px solid #475569;
      padding: 6px 14px;
      border-radius: 6px;
      cursor: pointer;
      font-weight: 500;
      font-size: 0.85rem;
      transition: all 0.2s ease;
    }
    button:hover { background: #475569; border-color: #64748b; }
    button:disabled { opacity: 0.4; cursor: not-allowed; }
    .slide-counter { font-size: 0.9rem; color: #94a3b8; font-variant-numeric: tabular-nums; }
    .main-stage {
      flex: 1;
      position: relative;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
      background: radial-gradient(circle at center, #1e293b 0%, #0f172a 100%);
    }
    .slide {
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
    }
    .slide.active { display: flex; }
    .slide-body {
      flex: 1;
      position: relative;
      overflow: hidden;
      background: #020617;
    }
    .slide-body img {
      width: 100%;
      height: 100%;
      object-fit: contain;
      object-position: top center;
    }
    .slide-footer {
      height: 48px;
      background: #0f172a;
      border-top: 1px solid #1e293b;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 20px;
      font-size: 0.85rem;
      color: #94a3b8;
    }
    .slide-footer strong { color: #f1f5f9; }
    .progress-bar {
      height: 4px;
      background: #1e293b;
      width: 100%;
    }
    .progress-fill {
      height: 100%;
      background: #38bdf8;
      width: 0%;
      transition: width 0.3s ease;
    }
  </style>
</head>
<body>
  <header>
    <div class="brand">
      🎓 Studyloop <span>App UI Presentation Deck</span>
    </div>
    <div class="controls">
      <button onclick="prevSlide()" id="prevBtn">← Previous</button>
      <span class="slide-counter" id="counter">Slide 1 of ${slides.length}</span>
      <button onclick="nextSlide()" id="nextBtn">Next →</button>
      <button onclick="toggleFullscreen()">⛶ Fullscreen</button>
    </div>
  </header>
  <div class="progress-bar"><div class="progress-fill" id="progress"></div></div>
  <div class="main-stage">
    ${slides.map((s, idx) => `
    <div class="slide ${idx === 0 ? 'active' : ''}" id="slide-${s.id}">
      <div class="slide-body">
        <img src="${s.image}" alt="${s.title}" />
      </div>
      <div class="slide-footer">
        <div><strong>${s.id}. ${s.title}</strong> — ${s.desc}</div>
        <div>Route: <code>${s.path}</code></div>
      </div>
    </div>
    `).join('')}
  </div>

  <script>
    let currentIndex = 0;
    const totalSlides = ${slides.length};

    function updateSlide() {
      document.querySelectorAll('.slide').forEach((el, idx) => {
        el.classList.toggle('active', idx === currentIndex);
      });
      document.getElementById('counter').innerText = \`Slide \${currentIndex + 1} of \${totalSlides}\`;
      document.getElementById('prevBtn').disabled = currentIndex === 0;
      document.getElementById('nextBtn').disabled = currentIndex === totalSlides - 1;
      document.getElementById('progress').style.width = \`\${((currentIndex + 1) / totalSlides) * 100}%\`;
    }

    function prevSlide() {
      if (currentIndex > 0) { currentIndex--; updateSlide(); }
    }
    function nextSlide() {
      if (currentIndex < totalSlides - 1) { currentIndex++; updateSlide(); }
    }
    function toggleFullscreen() {
      if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen();
      } else {
        document.exitFullscreen();
      }
    }

    window.addEventListener('keydown', (e) => {
      if (e.key === 'ArrowRight' || e.key === ' ') nextSlide();
      if (e.key === 'ArrowLeft') prevSlide();
      if (e.key.toLowerCase() === 'f') toggleFullscreen();
    });

    updateSlide();
  </script>
</body>
</html>`;
}

main().catch(console.error);
