package handlers

import (
	"fmt"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// Common CSS for PDF rendering (Tailwind-like inline styles since data URLs can't load external CSS)
const pdfBaseCSS = `
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; font-size: 12px; color: #1e293b; line-height: 1.4; padding: 20px; }
  .text-center { text-align: center; }
  .text-right { text-align: right; }
  .font-bold { font-weight: 700; }
  .font-semibold { font-weight: 600; }
  .font-mono { font-family: 'Courier New', monospace; }
  .text-xs { font-size: 10px; }
  .text-sm { font-size: 12px; }
  .text-base { font-size: 14px; }
  .text-lg { font-size: 16px; }
  .text-gray-400 { color: #9ca3af; }
  .text-gray-500 { color: #6b7280; }
  .text-gray-600 { color: #4b5563; }
  .text-gray-700 { color: #374151; }
  .text-gray-800 { color: #1f2937; }
  .text-gray-900 { color: #111827; }
  .mb-1 { margin-bottom: 4px; }
  .mb-2 { margin-bottom: 8px; }
  .mb-4 { margin-bottom: 16px; }
  .mb-6 { margin-bottom: 24px; }
  .mb-8 { margin-bottom: 32px; }
  .mt-1 { margin-top: 4px; }
  .mt-2 { margin-top: 8px; }
  .p-3 { padding: 12px; }
  .py-2 { padding-top: 8px; padding-bottom: 8px; }
  .px-3 { padding-left: 12px; padding-right: 12px; }
  .uppercase { text-transform: uppercase; }
  .tracking-wide { letter-spacing: 0.025em; }
  .tracking-widest { letter-spacing: 0.1em; }
  .border { border: 1px solid #e2e8f0; }
  .border-b { border-bottom: 1px solid #e2e8f0; }
  .border-t { border-top: 1px solid #e2e8f0; }
  .border-b-2 { border-bottom: 2px solid #1f2937; }
  .border-t-2 { border-top: 2px solid #1f2937; }
  .rounded { border-radius: 4px; }
  .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
  .flex-between { display: flex; justify-content: space-between; }
  .flex-end { display: flex; justify-content: flex-end; }
  .inline-block { display: inline-block; }
  table { width: 100%; border-collapse: collapse; }
  table th, table td { border: 1px solid #cbd5e1; padding: 6px 8px; font-size: 11px; vertical-align: top; }
  table th { background: #f1f5f9; font-weight: 600; text-align: center; font-size: 10px; text-transform: uppercase; letter-spacing: 0.025em; color: #334155; }
  table td { color: #1e293b; }
  .totals-row td { font-weight: 700; background: #f8fafc; }
  .summary-box { width: 260px; border: 1px solid #e2e8f0; border-radius: 4px; overflow: hidden; }
  .summary-row { display: flex; justify-content: space-between; padding: 6px 12px; border-bottom: 1px solid #f1f5f9; }
  .summary-total { display: flex; justify-content: space-between; padding: 8px 12px; background: #f8fafc; }
  .sig-section { display: grid; grid-template-columns: 1fr 1fr; gap: 32px; padding-top: 16px; border-top: 1px solid #e2e8f0; }
  .sig-block { }
  .sig-line { border-bottom: 1px solid #94a3b8; min-width: 180px; display: inline-block; height: 1px; margin-bottom: 2px; }
  @page { margin: 1cm; size: A4 portrait; }
</style>
`

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func formatINR(amount float64) string {
	return helpers.TemplateFuncs()["formatINR"].(func(float64) string)(amount)
}

func formatAddressHTML(addr *models.Address) string {
	if addr == nil {
		return ""
	}
	var parts []string
	for _, v := range addr.Data {
		v = strings.TrimSpace(v)
		if v != "" {
			parts = append(parts, fmt.Sprintf(`<p class="text-xs text-gray-600">%s</p>`, esc(v)))
		}
	}
	return strings.Join(parts, "\n")
}

func buildTransitPrintHTML(project *models.Project, dc *models.DeliveryChallan,
	transitDetails *models.DCTransitDetails, lineItems []models.DCLineItem,
	company *models.CompanySettings, shipToAddress, billToAddress *models.Address,
	totalTaxable, totalTax, grandTotal, roundedTotal, roundOff float64,
	totalQty int, halfTax float64, amountInWords string) string {

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8">`)
	b.WriteString(pdfBaseCSS)
	b.WriteString(`</head><body>`)

	// Company header
	if company != nil {
		b.WriteString(fmt.Sprintf(`<div class="text-center mb-6" style="border-bottom: 2px solid #1f2937; padding-bottom: 16px;">
			<h1 class="text-lg font-bold text-gray-900 uppercase tracking-wide">%s</h1>
			<p class="text-xs text-gray-600 mt-1">%s, %s, %s %s</p>
			<p class="text-xs text-gray-700 mt-1 font-semibold font-mono">GSTIN: %s</p>
		</div>`, esc(company.Name), esc(company.Address), esc(company.City), esc(company.State), esc(company.Pincode), esc(company.GSTIN)))
	}

	// DC Title
	b.WriteString(`<div class="text-center mb-6"><h2 class="text-base font-bold text-gray-900 uppercase tracking-widest border inline-block" style="padding: 6px 32px;">Delivery Challan</h2></div>`)

	// Two column header
	b.WriteString(`<div class="grid-2 mb-4">`)
	// Left
	b.WriteString(`<div class="border rounded p-3">`)
	b.WriteString(fmt.Sprintf(`<div class="flex-between text-xs mb-1"><span class="text-gray-500 font-semibold">DC No:</span><span class="font-semibold font-mono">%s</span></div>`, esc(dc.DCNumber)))
	if dc.ChallanDate != nil {
		b.WriteString(fmt.Sprintf(`<div class="flex-between text-xs mb-1"><span class="text-gray-500 font-semibold">Date:</span><span>%s</span></div>`, esc(*dc.ChallanDate)))
	}
	if transitDetails != nil {
		if transitDetails.TransporterName != "" {
			b.WriteString(fmt.Sprintf(`<div class="flex-between text-xs mb-1"><span class="text-gray-500 font-semibold">Transporter:</span><span>%s</span></div>`, esc(transitDetails.TransporterName)))
		}
		if transitDetails.VehicleNumber != "" {
			b.WriteString(fmt.Sprintf(`<div class="flex-between text-xs mb-1"><span class="text-gray-500 font-semibold">Vehicle:</span><span class="font-mono">%s</span></div>`, esc(transitDetails.VehicleNumber)))
		}
		if transitDetails.EwayBillNumber != "" {
			b.WriteString(fmt.Sprintf(`<div class="flex-between text-xs mb-1"><span class="text-gray-500 font-semibold">E-Way Bill:</span><span class="font-mono">%s</span></div>`, esc(transitDetails.EwayBillNumber)))
		}
	}
	b.WriteString(`<div class="flex-between text-xs"><span class="text-gray-500 font-semibold">Reverse Charge:</span><span>No</span></div>`)
	b.WriteString(`</div>`)
	// Right
	b.WriteString(`<div class="border rounded p-3">`)
	if shipToAddress != nil {
		if state, ok := shipToAddress.Data["state"]; ok && state != "" {
			b.WriteString(fmt.Sprintf(`<div class="flex-between text-xs mb-1"><span class="text-gray-500 font-semibold">State:</span><span>%s</span></div>`, esc(state)))
		}
		if city, ok := shipToAddress.Data["city"]; ok && city != "" {
			b.WriteString(fmt.Sprintf(`<div class="flex-between text-xs"><span class="text-gray-500 font-semibold">Place of Supply:</span><span>%s</span></div>`, esc(city)))
		}
	}
	b.WriteString(`</div></div>`)

	// PO Details
	if project != nil {
		b.WriteString(`<div class="border rounded p-3 mb-4 text-xs">`)
		if project.POReference != "" {
			b.WriteString(fmt.Sprintf(`<div class="flex-between mb-1"><span class="text-gray-500 font-semibold">PO Number:</span><span class="font-semibold font-mono">%s</span></div>`, esc(project.POReference)))
		}
		if project.PODate != nil {
			b.WriteString(fmt.Sprintf(`<div class="flex-between mb-1"><span class="text-gray-500 font-semibold">PO Date:</span><span>%s</span></div>`, esc(*project.PODate)))
		}
		b.WriteString(fmt.Sprintf(`<div class="flex-between"><span class="text-gray-500 font-semibold">Project:</span><span>%s</span></div>`, esc(project.Name)))
		b.WriteString(`</div>`)
	}

	// Address blocks
	b.WriteString(`<div class="grid-2 mb-6">`)
	// Bill From
	if company != nil {
		b.WriteString(fmt.Sprintf(`<div class="border rounded p-3"><p class="text-xs font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Bill From</p><p class="text-xs font-semibold">%s</p><p class="text-xs text-gray-600">%s, %s, %s %s</p><p class="text-xs text-gray-600 mt-1"><span class="font-semibold">GSTIN:</span> %s</p></div>`,
			esc(company.Name), esc(company.Address), esc(company.City), esc(company.State), esc(company.Pincode), esc(company.GSTIN)))
	}
	// Bill To
	b.WriteString(`<div class="border rounded p-3"><p class="text-xs font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Bill To</p>`)
	if billToAddress != nil {
		b.WriteString(formatAddressHTML(billToAddress))
	} else {
		b.WriteString(`<p class="text-xs text-gray-400">Not specified</p>`)
	}
	b.WriteString(`</div>`)
	// Dispatch From
	if company != nil {
		b.WriteString(fmt.Sprintf(`<div class="border rounded p-3"><p class="text-xs font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Dispatch From</p><p class="text-xs font-semibold">%s</p><p class="text-xs text-gray-600">%s, %s, %s %s</p></div>`,
			esc(company.Name), esc(company.Address), esc(company.City), esc(company.State), esc(company.Pincode)))
	}
	// Ship To
	b.WriteString(`<div class="border rounded p-3"><p class="text-xs font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Ship To</p>`)
	if shipToAddress != nil {
		b.WriteString(formatAddressHTML(shipToAddress))
	}
	b.WriteString(`</div></div>`)

	// Product table
	b.WriteString(`<div class="mb-6"><table>`)
	b.WriteString(`<thead><tr>
		<th style="width:36px;">S.No</th>
		<th style="width:130px;">Item Description</th>
		<th style="width:100px;">Serial Nos</th>
		<th style="width:36px;">UoM</th>
		<th style="width:50px;">HSN Code</th>
		<th style="width:36px;">Qty</th>
		<th style="width:75px;">Per Unit Price</th>
		<th style="width:75px;">Taxable Value</th>
		<th style="width:36px;">GST %</th>
		<th style="width:70px;">GST Amount</th>
		<th style="width:75px;">Total Value</th>
	</tr></thead><tbody>`)

	for i, li := range lineItems {
		serialsHTML := ""
		for _, s := range li.SerialNumbers {
			serialsHTML += esc(s) + "<br>"
		}
		desc := esc(li.ItemName)
		if li.BrandModel != "" {
			desc += "<br><span class='text-gray-500'>" + esc(li.BrandModel) + "</span>"
		}
		if li.ItemDescription != "" {
			desc += "<br><span class='text-gray-500'>" + esc(li.ItemDescription) + "</span>"
		}
		b.WriteString(fmt.Sprintf(`<tr>
			<td class="text-center">%d</td>
			<td>%s</td>
			<td class="font-mono" style="font-size:10px;">%s</td>
			<td class="text-center">%s</td>
			<td class="text-center font-mono">%s</td>
			<td class="text-center">%d</td>
			<td class="text-right font-mono">&#8377;%s</td>
			<td class="text-right font-mono">&#8377;%s</td>
			<td class="text-center">%.0f%%</td>
			<td class="text-right font-mono">&#8377;%s</td>
			<td class="text-right font-mono font-semibold">&#8377;%s</td>
		</tr>`, i+1, desc, serialsHTML, esc(li.UoM), esc(li.HSNCode), li.Quantity,
			formatINR(li.Rate), formatINR(li.TaxableAmount), li.TaxPercentage,
			formatINR(li.TaxAmount), formatINR(li.TotalAmount)))
	}

	// Totals row
	b.WriteString(fmt.Sprintf(`<tr class="totals-row">
		<td colspan="2"></td>
		<td class="text-center font-bold uppercase" style="font-size:10px;">Total</td>
		<td></td><td></td>
		<td class="text-center font-mono">%d</td>
		<td></td>
		<td class="text-right font-mono">&#8377;%s</td>
		<td></td>
		<td class="text-right font-mono">&#8377;%s</td>
		<td class="text-right font-mono">&#8377;%s</td>
	</tr>`, totalQty, formatINR(totalTaxable), formatINR(totalTax), formatINR(grandTotal)))
	b.WriteString(`</tbody></table></div>`)

	// Tax summary
	cgst := helpers.TemplateFuncs()["formatINR"].(func(float64) string)(halfTax)
	sgst := cgst
	b.WriteString(`<div class="flex-end mb-6"><div class="summary-box">`)
	b.WriteString(fmt.Sprintf(`<div class="summary-row"><span class="text-gray-500">Taxable Value</span><span class="font-semibold font-mono">&#8377;%s</span></div>`, formatINR(totalTaxable)))
	b.WriteString(fmt.Sprintf(`<div class="summary-row"><span class="text-gray-500">CGST</span><span class="font-semibold font-mono">&#8377;%s</span></div>`, cgst))
	b.WriteString(fmt.Sprintf(`<div class="summary-row"><span class="text-gray-500">SGST</span><span class="font-semibold font-mono">&#8377;%s</span></div>`, sgst))
	b.WriteString(fmt.Sprintf(`<div class="summary-row"><span class="text-gray-500">Round Off</span><span class="font-semibold font-mono">&#8377;%.2f</span></div>`, roundOff))
	b.WriteString(fmt.Sprintf(`<div class="summary-total"><span class="font-bold">Invoice Value</span><span class="font-bold font-mono">&#8377;%s</span></div>`, formatINR(roundedTotal)))
	b.WriteString(`</div></div>`)

	// Amount in words
	b.WriteString(fmt.Sprintf(`<div class="border rounded p-3 mb-6"><p class="font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Amount in Words</p><p class="text-xs font-semibold">%s</p></div>`, esc(amountInWords)))

	// Notes
	if transitDetails != nil && transitDetails.Notes != "" {
		b.WriteString(fmt.Sprintf(`<div class="border rounded p-3 mb-8"><p class="font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Notes</p><p class="text-xs text-gray-700">%s</p></div>`, esc(transitDetails.Notes)))
	}

	// Signatures
	b.WriteString(`<div class="sig-section">`)
	b.WriteString(`<div><p class="text-xs font-semibold text-gray-700 mb-8">Receiver's Signature</p><div class="sig-line"></div><p class="text-xs text-gray-400 mt-1">Name: _________________________</p></div>`)
	b.WriteString(`<div style="text-align:right;">`)
	if company != nil {
		b.WriteString(fmt.Sprintf(`<p class="text-xs font-semibold text-gray-700 mb-2">For %s</p>`, esc(company.Name)))
	}
	b.WriteString(`<div style="height:48px;margin-bottom:8px;"></div><p class="text-xs font-semibold text-gray-600">Authorised Signatory</p>`)
	b.WriteString(`</div></div>`)

	b.WriteString(`</body></html>`)
	return b.String()
}

func buildOfficialPrintHTML(project *models.Project, dc *models.DeliveryChallan,
	lineItems []models.DCLineItem, company *models.CompanySettings,
	shipToAddress, billToAddress *models.Address, totalQty int) string {

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8">`)
	b.WriteString(pdfBaseCSS)
	b.WriteString(`</head><body>`)

	// Company header
	if company != nil {
		b.WriteString(fmt.Sprintf(`<div class="text-center mb-6">
			<h1 class="text-lg font-bold text-gray-900 uppercase tracking-wide">%s</h1>
			<p class="text-xs text-gray-600 mt-1">%s, %s, %s %s</p>`, esc(company.Name), esc(company.Address), esc(company.City), esc(company.State), esc(company.Pincode)))
		if company.Email != "" {
			b.WriteString(fmt.Sprintf(`<p class="text-xs text-gray-500 mt-1">Email: <span class="font-semibold text-gray-700">%s</span></p>`, esc(company.Email)))
		}
		b.WriteString(fmt.Sprintf(`<div class="text-xs text-gray-500 mt-1" style="display:flex;justify-content:center;gap:16px;">
			<span>GSTIN: <span class="font-mono font-semibold text-gray-700">%s</span></span>`, esc(company.GSTIN)))
		if company.CIN != "" {
			b.WriteString(fmt.Sprintf(`<span>CIN: <span class="font-mono font-semibold text-gray-700">%s</span></span>`, esc(company.CIN)))
		}
		b.WriteString(`</div></div>`)
	}

	// DC Title
	b.WriteString(`<div style="border-top:2px solid #1f2937;border-bottom:2px solid #1f2937;padding:8px 0;" class="text-center mb-6"><h2 class="text-base font-bold uppercase tracking-widest">Delivery Challan</h2></div>`)

	// Copy indicators
	b.WriteString(`<div style="display:flex;justify-content:flex-end;gap:24px;margin-bottom:16px;" class="text-xs text-gray-600">
		<span>&#9745; Original</span>
		<span>&#9745; Duplicate</span>
		<span>&#9745; Triplicate</span>
	</div>`)

	// DC meta
	b.WriteString(`<div class="grid-2 mb-6 text-sm">`)
	b.WriteString(fmt.Sprintf(`<div><span class="text-gray-500 font-semibold">DC No: </span><span class="font-mono font-semibold">%s</span></div>`, esc(dc.DCNumber)))
	if dc.ChallanDate != nil {
		b.WriteString(fmt.Sprintf(`<div><span class="text-gray-500 font-semibold">Date: </span><span class="font-semibold">%s</span></div>`, esc(*dc.ChallanDate)))
	}
	if shipToAddress != nil {
		if mandal, ok := shipToAddress.Data["mandal"]; ok && mandal != "" {
			b.WriteString(fmt.Sprintf(`<div><span class="text-gray-500 font-semibold">Mandal/ULB: </span><span class="font-semibold">%s</span></div>`, esc(mandal)))
		}
		if code, ok := shipToAddress.Data["mandal_code"]; ok && code != "" {
			b.WriteString(fmt.Sprintf(`<div><span class="text-gray-500 font-semibold">Mandal Code: </span><span class="font-mono font-semibold">%s</span></div>`, esc(code)))
		}
	}
	b.WriteString(`</div>`)

	// Reference info
	if project != nil {
		b.WriteString(`<div class="mb-6 text-sm">`)
		if project.TenderRefNumber != "" {
			b.WriteString(fmt.Sprintf(`<div class="mb-1"><span class="text-gray-500 font-semibold">Tender Ref: </span><span>%s</span></div>`, esc(project.TenderRefNumber)))
		}
		if project.POReference != "" {
			po := esc(project.POReference)
			if project.PODate != nil {
				po += fmt.Sprintf(" (%s)", esc(*project.PODate))
			}
			b.WriteString(fmt.Sprintf(`<div class="mb-1"><span class="text-gray-500 font-semibold">PO Ref: </span><span>%s</span></div>`, po))
		}
		b.WriteString(fmt.Sprintf(`<div class="mb-1"><span class="text-gray-500 font-semibold">Project: </span><span>%s</span></div>`, esc(project.Name)))
		b.WriteString(`</div>`)
	}

	// Purpose
	if project != nil && project.PurposeText != "" {
		b.WriteString(fmt.Sprintf(`<div class="mb-4 text-sm"><span class="text-gray-500 font-semibold">Purpose: </span><span class="font-semibold uppercase">%s</span></div>`, esc(project.PurposeText)))
	}

	// Issued To
	if shipToAddress != nil {
		issuedTo := ""
		if d, ok := shipToAddress.Data["district"]; ok && d != "" {
			issuedTo = d + " District"
		}
		if m, ok := shipToAddress.Data["mandal"]; ok && m != "" {
			if issuedTo != "" {
				issuedTo += ", "
			}
			issuedTo += m + " Mandal/ULB"
		}
		if issuedTo != "" {
			b.WriteString(fmt.Sprintf(`<div class="mb-4 text-sm"><span class="text-gray-500 font-semibold">Issued To: </span><span class="font-semibold">%s</span></div>`, esc(issuedTo)))
		}
	}

	// Addresses
	b.WriteString(`<div class="grid-2 mb-6">`)
	if billToAddress != nil {
		b.WriteString(`<div class="border rounded p-3"><p class="font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Bill To</p>`)
		b.WriteString(formatAddressHTML(billToAddress))
		b.WriteString(`</div>`)
	}
	if shipToAddress != nil {
		b.WriteString(`<div class="border rounded p-3"><p class="font-bold text-gray-400 uppercase mb-1" style="font-size:9px;">Ship To</p>`)
		b.WriteString(formatAddressHTML(shipToAddress))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)

	// Product table (no pricing)
	b.WriteString(`<div class="mb-8"><table>`)
	b.WriteString(`<thead><tr>
		<th style="width:40px;">S.No</th>
		<th>Item Name</th>
		<th>Description</th>
		<th>Brand / Model No</th>
		<th style="width:50px;">Qty</th>
		<th>Serial Number</th>
		<th style="width:70px;">Remarks</th>
	</tr></thead><tbody>`)

	for i, li := range lineItems {
		serialsHTML := ""
		for _, s := range li.SerialNumbers {
			serialsHTML += esc(s) + "<br>"
		}
		b.WriteString(fmt.Sprintf(`<tr>
			<td class="text-center font-semibold">%d</td>
			<td class="font-semibold">%s</td>
			<td class="text-gray-700">%s</td>
			<td class="font-mono text-xs text-gray-700">%s</td>
			<td class="text-center font-semibold">%d</td>
			<td class="font-mono text-xs text-gray-700">%s</td>
			<td class="text-gray-400 text-center">&mdash;</td>
		</tr>`, i+1, esc(li.ItemName), esc(li.ItemDescription), esc(li.BrandModel), li.Quantity, serialsHTML))
	}
	b.WriteString(`</tbody></table></div>`)

	// Acknowledgement
	b.WriteString(`<div class="border rounded p-3 mb-8" style="background:#fafafa;">
		<p class="text-sm text-gray-800 font-semibold mb-2" style="font-style:italic;">"It is certified that the material is received in good condition."</p>
		<div class="text-sm"><span class="text-gray-500 font-semibold">Date of Receipt: </span><span style="border-bottom:1px solid #9ca3af;min-width:200px;display:inline-block;">&nbsp;</span></div>
	</div>`)

	// Dual signature block
	b.WriteString(`<div class="sig-section">`)
	// Left: FSSPL Rep
	b.WriteString(`<div class="text-center">
		<p class="font-bold uppercase text-xs mb-6" style="letter-spacing:0.05em;">FSSPL Representative</p>
		<div style="width:180px;height:60px;border:1px dashed #d1d5db;border-radius:4px;margin:0 auto 16px;display:flex;align-items:center;justify-content:center;"><span class="text-xs text-gray-400">Signature</span></div>
		<div style="text-align:left;" class="text-xs">
			<div class="mb-1">Name: <span style="border-bottom:1px solid #9ca3af;display:inline-block;min-width:140px;">&nbsp;</span></div>
			<div class="mb-1">Designation: <span style="border-bottom:1px solid #9ca3af;display:inline-block;min-width:120px;">&nbsp;</span></div>
			<div>Mobile: <span style="border-bottom:1px solid #9ca3af;display:inline-block;min-width:130px;">&nbsp;</span></div>
		</div>
	</div>`)
	// Right: Dept Official
	b.WriteString(`<div class="text-center">
		<p class="font-bold uppercase text-xs mb-6" style="letter-spacing:0.05em;">Department Official</p>
		<div style="width:180px;height:60px;border:1px dashed #d1d5db;border-radius:4px;margin:0 auto 16px;display:flex;align-items:center;justify-content:center;"><span class="text-xs text-gray-400">Signature with Seal &amp; Date</span></div>
		<div style="text-align:left;" class="text-xs">
			<div class="mb-1">Name: <span style="border-bottom:1px solid #9ca3af;display:inline-block;min-width:140px;">&nbsp;</span></div>
			<div class="mb-1">Designation: <span style="border-bottom:1px solid #9ca3af;display:inline-block;min-width:120px;">&nbsp;</span></div>
			<div>Mobile: <span style="border-bottom:1px solid #9ca3af;display:inline-block;min-width:130px;">&nbsp;</span></div>
		</div>
	</div>`)
	b.WriteString(`</div>`)

	b.WriteString(`</body></html>`)
	return b.String()
}
