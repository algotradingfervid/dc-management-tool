package services

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// MergePDFs combines multiple PDF byte slices into a single PDF document.
// Each input PDF starts on a new page in the merged output.
func MergePDFs(pdfs [][]byte) ([]byte, error) {
	if len(pdfs) == 0 {
		return nil, fmt.Errorf("no PDFs to merge")
	}
	if len(pdfs) == 1 {
		return pdfs[0], nil
	}

	readers := make([]io.ReadSeeker, len(pdfs))
	for i, pdf := range pdfs {
		readers[i] = bytes.NewReader(pdf)
	}

	var buf bytes.Buffer
	conf := model.NewDefaultConfiguration()
	if err := api.MergeRaw(readers, &buf, false, conf); err != nil {
		return nil, fmt.Errorf("pdf merge error: %w", err)
	}
	return buf.Bytes(), nil
}
