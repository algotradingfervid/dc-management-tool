package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// GeneratePDFFromHTML renders an HTML string to PDF using headless Chrome.
func GeneratePDFFromHTML(html string) ([]byte, error) {
	// Encode HTML as a data URL
	encoded := base64.StdEncoding.EncodeToString([]byte(html))
	dataURL := "data:text/html;base64," + encoded
	return GeneratePDFFromURL(dataURL)
}

// GeneratePDFFromURL navigates to the given URL using headless Chrome and returns the page as PDF bytes.
func GeneratePDFFromURL(targetURL string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte

	if err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body"),
		// Small delay for CSS/fonts to load
		chromedp.Sleep(500*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			return err
		}),
	); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf, nil
}

// BuildPrintURL constructs the print view URL for a DC on the local server.
func BuildPrintURL(baseURL string, projectID, dcID int, dcType string) string {
	var path string
	if dcType == "official" {
		path = fmt.Sprintf("/projects/%d/dcs/%d/official-print", projectID, dcID)
	} else {
		path = fmt.Sprintf("/projects/%d/dcs/%d/print", projectID, dcID)
	}
	u, _ := url.Parse(baseURL)
	u.Path = path
	return u.String()
}
