package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	pdfreader "github.com/ledongthuc/pdf"
)

type ExtractionResult struct {
	StrategyName string
	Duration     time.Duration
	Pages        int
	Words        int
	Chars        int
	Throughput   float64
	Speedup      float64
	Error        string
}

type PageOutput struct {
	PageNum int
	Text    string
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// -------------------------------------------------------------
// Strategy 1: Sequential Page-by-Page (Single Reader)
// -------------------------------------------------------------
func extractSequentialSingleReader(filePath string, startPage, endPage int) (string, error) {
	file, reader, err := pdfreader.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	total := reader.NumPage()
	minP := 1
	if startPage > 0 {
		minP = startPage
	}
	maxP := total
	if endPage > 0 && endPage < total {
		maxP = endPage
	}

	var sb strings.Builder
	for p := minP; p <= maxP; p++ {
		page := reader.Page(p)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("page %d: %w", p, err)
		}
		norm := normalizeWhitespace(text)
		if norm != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(norm)
		}
	}
	return sb.String(), nil
}

// -------------------------------------------------------------
// Strategy 2: Sequential Stream Reader (reader.GetPlainText())
// -------------------------------------------------------------
func extractSequentialStream(filePath string) (string, error) {
	file, reader, err := pdfreader.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	plainReader, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, plainReader); err != nil {
		return "", err
	}

	return normalizeWhitespace(buf.String()), nil
}

// -------------------------------------------------------------
// Strategy 3: Parallel Fixed Workers (Shared Reader)
// -------------------------------------------------------------
func extractParallelSharedReader(filePath string, startPage, endPage, numWorkers int) (string, error) {
	file, reader, err := pdfreader.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	total := reader.NumPage()
	minP := 1
	if startPage > 0 {
		minP = startPage
	}
	maxP := total
	if endPage > 0 && endPage < total {
		maxP = endPage
	}

	pagesCount := maxP - minP + 1
	results := make([]PageOutput, pagesCount)

	type task struct {
		idx     int
		pageNum int
	}

	tasks := make(chan task, pagesCount)
	for i := 0; i < pagesCount; i++ {
		tasks <- task{idx: i, pageNum: minP + i}
	}
	close(tasks)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var workerErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				page := reader.Page(t.pageNum)
				if page.V.IsNull() {
					continue
				}
				text, err := page.GetPlainText(nil)
				if err != nil {
					errOnce.Do(func() {
						workerErr = fmt.Errorf("page %d: %w", t.pageNum, err)
					})
					return
				}
				results[t.idx] = PageOutput{
					PageNum: t.pageNum,
					Text:    normalizeWhitespace(text),
				}
			}
		}()
	}

	wg.Wait()
	if workerErr != nil {
		return "", workerErr
	}

	var sb strings.Builder
	for _, res := range results {
		if res.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(res.Text)
		}
	}
	return sb.String(), nil
}

// -------------------------------------------------------------
// Strategy 4: Parallel Worker Pool (Independent File Handle per Worker)
// -------------------------------------------------------------
func extractParallelIndependentHandles(filePath string, startPage, endPage, numWorkers int) (string, error) {
	initFile, initReader, err := pdfreader.Open(filePath)
	if err != nil {
		return "", err
	}
	total := initReader.NumPage()
	initFile.Close()

	minP := 1
	if startPage > 0 {
		minP = startPage
	}
	maxP := total
	if endPage > 0 && endPage < total {
		maxP = endPage
	}

	pagesCount := maxP - minP + 1
	results := make([]PageOutput, pagesCount)

	type task struct {
		idx     int
		pageNum int
	}

	tasks := make(chan task, pagesCount)
	for i := 0; i < pagesCount; i++ {
		tasks <- task{idx: i, pageNum: minP + i}
	}
	close(tasks)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var workerErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, r, err := pdfreader.Open(filePath)
			if err != nil {
				errOnce.Do(func() { workerErr = err })
				return
			}
			defer f.Close()

			for t := range tasks {
				page := r.Page(t.pageNum)
				if page.V.IsNull() {
					continue
				}
				text, err := page.GetPlainText(nil)
				if err != nil {
					errOnce.Do(func() {
						workerErr = fmt.Errorf("page %d: %w", t.pageNum, err)
					})
					return
				}
				results[t.idx] = PageOutput{
					PageNum: t.pageNum,
					Text:    normalizeWhitespace(text),
				}
			}
		}()
	}

	wg.Wait()
	if workerErr != nil {
		return "", workerErr
	}

	var sb strings.Builder
	for _, res := range results {
		if res.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(res.Text)
		}
	}
	return sb.String(), nil
}

// -------------------------------------------------------------
// Strategy 5: In-Memory Preload + Parallel Readers (Zero Disk I/O Bottleneck)
// -------------------------------------------------------------
func extractParallelInMemory(filePath string, startPage, endPage, numWorkers int) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	bytesReader := bytes.NewReader(data)
	initReader, err := pdfreader.NewReader(bytesReader, int64(len(data)))
	if err != nil {
		return "", err
	}

	total := initReader.NumPage()
	minP := 1
	if startPage > 0 {
		minP = startPage
	}
	maxP := total
	if endPage > 0 && endPage < total {
		maxP = endPage
	}

	pagesCount := maxP - minP + 1
	results := make([]PageOutput, pagesCount)

	type task struct {
		idx     int
		pageNum int
	}

	tasks := make(chan task, pagesCount)
	for i := 0; i < pagesCount; i++ {
		tasks <- task{idx: i, pageNum: minP + i}
	}
	close(tasks)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var workerErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rdr := bytes.NewReader(data)
			r, err := pdfreader.NewReader(rdr, int64(len(data)))
			if err != nil {
				errOnce.Do(func() { workerErr = err })
				return
			}

			for t := range tasks {
				page := r.Page(t.pageNum)
				if page.V.IsNull() {
					continue
				}
				text, err := page.GetPlainText(nil)
				if err != nil {
					errOnce.Do(func() {
						workerErr = fmt.Errorf("page %d: %w", t.pageNum, err)
					})
					return
				}
				results[t.idx] = PageOutput{
					PageNum: t.pageNum,
					Text:    normalizeWhitespace(text),
				}
			}
		}()
	}

	wg.Wait()
	if workerErr != nil {
		return "", workerErr
	}

	var sb strings.Builder
	for _, res := range results {
		if res.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(res.Text)
		}
	}
	return sb.String(), nil
}

// -------------------------------------------------------------
// Strategy 6: Dynamic / Adaptive Chunk Range Allocation
// -------------------------------------------------------------
func extractDynamicAdaptiveChunks(filePath string, startPage, endPage, numWorkers int) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	bytesReader := bytes.NewReader(data)
	initReader, err := pdfreader.NewReader(bytesReader, int64(len(data)))
	if err != nil {
		return "", err
	}

	total := initReader.NumPage()
	minP := 1
	if startPage > 0 {
		minP = startPage
	}
	maxP := total
	if endPage > 0 && endPage < total {
		maxP = endPage
	}

	totalPages := maxP - minP + 1

	// Build adaptive chunks: larger chunks first, decaying to smaller chunks at the tail
	type pageChunk struct {
		chunkID   int
		startPage int
		endPage   int
	}

	var chunks []pageChunk
	curr := minP
	chunkSize := totalPages / (numWorkers * 2)
	if chunkSize < 1 {
		chunkSize = 1
	}
	if chunkSize > 8 {
		chunkSize = 8
	}

	cID := 0
	for curr <= maxP {
		remaining := maxP - curr + 1
		take := chunkSize
		if take > remaining {
			take = remaining
		}
		chunks = append(chunks, pageChunk{
			chunkID:   cID,
			startPage: curr,
			endPage:   curr + take - 1,
		})
		cID++
		curr += take

		// Decaying tail strategy
		if chunkSize > 1 && remaining < (totalPages/3) {
			chunkSize = chunkSize / 2
			if chunkSize < 1 {
				chunkSize = 1
			}
		}
	}

	chunkResults := make([]string, len(chunks))
	taskCh := make(chan pageChunk, len(chunks))
	for _, c := range chunks {
		taskCh <- c
	}
	close(taskCh)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var workerErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rdr := bytes.NewReader(data)
			r, err := pdfreader.NewReader(rdr, int64(len(data)))
			if err != nil {
				errOnce.Do(func() { workerErr = err })
				return
			}

			for ch := range taskCh {
				var sb strings.Builder
				for p := ch.startPage; p <= ch.endPage; p++ {
					page := r.Page(p)
					if page.V.IsNull() {
						continue
					}
					text, err := page.GetPlainText(nil)
					if err != nil {
						errOnce.Do(func() {
							workerErr = fmt.Errorf("page %d: %w", p, err)
						})
						return
					}
					norm := normalizeWhitespace(text)
					if norm != "" {
						if sb.Len() > 0 {
							sb.WriteString("\n\n")
						}
						sb.WriteString(norm)
					}
				}
				chunkResults[ch.chunkID] = sb.String()
			}
		}()
	}

	wg.Wait()
	if workerErr != nil {
		return "", workerErr
	}

	var finalSb strings.Builder
	for _, cr := range chunkResults {
		if cr != "" {
			if finalSb.Len() > 0 {
				finalSb.WriteString("\n\n")
			}
			finalSb.WriteString(cr)
		}
	}
	return finalSb.String(), nil
}

func main() {
	pdfPath := flag.String("pdf", "learning go.pdf", "Path to target PDF file")
	startPage := flag.Int("start-page", 1, "Start page (1-based)")
	endPage := flag.Int("end-page", 30, "End page (1-based)")
	runs := flag.Int("runs", 2, "Number of benchmark runs per strategy")
	flag.Parse()

	targetPath := *pdfPath
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		candidates, _ := filepath.Glob("dev_data/uploads/*.pdf")
		if len(candidates) > 0 {
			targetPath = candidates[0]
			fmt.Printf("Notice: '%s' not found, using fallback: %s\n", *pdfPath, targetPath)
		} else {
			fmt.Fprintf(os.Stderr, "Error: PDF file '%s' not found.\n", *pdfPath)
			os.Exit(1)
		}
	}

	// Inspect PDF info
	f, r, err := pdfreader.Open(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	totalDocPages := r.NumPage()
	f.Close()

	minP := *startPage
	if minP <= 0 {
		minP = 1
	}
	maxP := *endPage
	if maxP <= 0 || maxP > totalDocPages {
		maxP = totalDocPages
	}
	if minP > maxP {
		minP = 1
	}
	numPages := maxP - minP + 1
	cpuCount := runtime.NumCPU()

	fmt.Println(strings.Repeat("=", 90))
	fmt.Println("GO-BASED PDF TEXT EXTRACTION STRATEGIES BENCHMARK")
	fmt.Printf("Target File  : %s\n", targetPath)
	fmt.Printf("Page Range   : %d to %d (Total Pages: %d of %d)\n", minP, maxP, numPages, totalDocPages)
	fmt.Printf("Host System  : %d Logical CPU Cores (%s/%s)\n", cpuCount, runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Iterations   : %d per strategy\n", *runs)
	fmt.Println(strings.Repeat("=", 90))

	type strategy struct {
		name string
		fn   func() (string, error)
	}

	strategies := []strategy{
		{
			name: "1. Sequential Page-by-Page (Single Reader)",
			fn:   func() (string, error) { return extractSequentialSingleReader(targetPath, minP, maxP) },
		},
		{
			name: "2. Sequential Stream Reader (reader.GetPlainText)",
			fn:   func() (string, error) { return extractSequentialStream(targetPath) },
		},
		{
			name: fmt.Sprintf("3. Parallel Shared Reader (%d Workers)", cpuCount),
			fn:   func() (string, error) { return extractParallelSharedReader(targetPath, minP, maxP, cpuCount) },
		},
		{
			name: fmt.Sprintf("4. Parallel Independent Handles (2 Workers)"),
			fn:   func() (string, error) { return extractParallelIndependentHandles(targetPath, minP, maxP, 2) },
		},
		{
			name: fmt.Sprintf("5. Parallel Independent Handles (4 Workers)"),
			fn:   func() (string, error) { return extractParallelIndependentHandles(targetPath, minP, maxP, 4) },
		},
		{
			name: fmt.Sprintf("6. Parallel Independent Handles (%d Workers, CPU Count)", cpuCount),
			fn:   func() (string, error) { return extractParallelIndependentHandles(targetPath, minP, maxP, cpuCount) },
		},
		{
			name: fmt.Sprintf("7. In-Memory Preload + Parallel Readers (4 Workers)"),
			fn:   func() (string, error) { return extractParallelInMemory(targetPath, minP, maxP, 4) },
		},
		{
			name: fmt.Sprintf("8. In-Memory Preload + Parallel Readers (%d Workers, CPU Count)", cpuCount),
			fn:   func() (string, error) { return extractParallelInMemory(targetPath, minP, maxP, cpuCount) },
		},
		{
			name: fmt.Sprintf("9. In-Memory Preload + Overcommit Readers (%d Workers)", cpuCount*2),
			fn:   func() (string, error) { return extractParallelInMemory(targetPath, minP, maxP, cpuCount*2) },
		},
		{
			name: fmt.Sprintf("10. Dynamic / Adaptive Decaying Chunks (%d Workers)", cpuCount),
			fn:   func() (string, error) { return extractDynamicAdaptiveChunks(targetPath, minP, maxP, cpuCount) },
		},
	}

	var results []ExtractionResult
	var baselineDuration time.Duration

	for idx, s := range strategies {
		fmt.Printf("\n[%d/%d] Running: %s ... ", idx+1, len(strategies), s.name)
		var totalDur time.Duration
		var textOut string
		var runErr error

		for r := 0; r < *runs; r++ {
			t0 := time.Now()
			out, err := s.fn()
			dur := time.Since(t0)
			if err != nil {
				runErr = err
				break
			}
			totalDur += dur
			textOut = out
		}

		if runErr != nil {
			fmt.Printf("FAILED (%v)\n", runErr)
			results = append(results, ExtractionResult{
				StrategyName: s.name,
				Error:        runErr.Error(),
			})
			continue
		}

		avgDur := totalDur / time.Duration(*runs)
		if idx == 0 {
			baselineDuration = avgDur
		}

		speedup := 1.0
		if baselineDuration > 0 && avgDur > 0 {
			speedup = float64(baselineDuration) / float64(avgDur)
		}

		words := len(strings.Fields(textOut))
		chars := len(textOut)
		throughput := float64(numPages) / avgDur.Seconds()

		fmt.Printf("Done in %v (%.1f p/s, %.2fx speedup)\n", avgDur.Round(time.Millisecond), throughput, speedup)

		results = append(results, ExtractionResult{
			StrategyName: s.name,
			Duration:     avgDur,
			Pages:        numPages,
			Words:        words,
			Chars:        chars,
			Throughput:   throughput,
			Speedup:      speedup,
		})
	}

	// Print Summary Table
	fmt.Println("\n" + strings.Repeat("=", 105))
	fmt.Println("BENCHMARK RESULTS SUMMARY (GO-BASED PDF EXTRACTION)")
	fmt.Println(strings.Repeat("=", 105))
	fmt.Printf("%-56s | %-12s | %-10s | %-9s | %-8s\n",
		"Strategy / Concurrency Mode", "Time", "Pages/sec", "Speedup", "Words")
	fmt.Println(strings.Repeat("-", 105))

	var bestDuration time.Duration = 1<<63 - 1
	for _, r := range results {
		if r.Error == "" && r.Duration < bestDuration && r.Duration > 0 {
			bestDuration = r.Duration
		}
	}

	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("%-56s | %-12s | %-10s | %-9s | %-8s (ERROR)\n",
				r.StrategyName, "ERR", "-", "-", "-")
			continue
		}
		isBest := ""
		if r.Duration == bestDuration {
			isBest = " 🏆"
		}
		fmt.Printf("%-56s | %-12v | %9.1f | %8.2fx | %8d%s\n",
			r.StrategyName, r.Duration.Round(time.Millisecond), r.Throughput, r.Speedup, r.Words, isBest)
	}
	fmt.Println(strings.Repeat("=", 105))

	// Sort and display top recommendations
	var validResults []ExtractionResult
	for _, r := range results {
		if r.Error == "" {
			validResults = append(validResults, r)
		}
	}
	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].Duration < validResults[j].Duration
	})

	if len(validResults) > 0 {
		fmt.Printf("\n🏆 Fastest Strategy: %s (%v, %.2fx speedup)\n\n",
			validResults[0].StrategyName, validResults[0].Duration.Round(time.Millisecond), validResults[0].Speedup)
	}
}
