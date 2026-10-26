package infrastructure

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// AnnualAccountsPDFRenderer renders Cabinet SGCAM-style Comptes Annuels and CERFA Liasse Fiscale HTML/PDF documents.
type AnnualAccountsPDFRenderer struct {
	pdfEngine templating.PdfEngine
}

// NewAnnualAccountsPDFRenderer constructs an AnnualAccountsPDFRenderer backed by pdfEngine.
func NewAnnualAccountsPDFRenderer(pdfEngine templating.PdfEngine) *AnnualAccountsPDFRenderer {
	if pdfEngine == nil {
		pdfEngine = templating.NewGotenbergPdfEngine("")
	}
	return &AnnualAccountsPDFRenderer{pdfEngine: pdfEngine}
}

// RenderAnnualAccountsHTML generates the full Comptes Annuels & Liasse Fiscale HTML document,
// excluding all disabled lines (`line.Disabled == true`) and displaying dynamically recomputed totals and SIG.
func (r *AnnualAccountsPDFRenderer) RenderAnnualAccountsHTML(
	meta domain.AnnualAccountsMetadata,
	stmt domain.ComputedFinancialStatementsDTO,
) (string, error) {
	entityName := strings.TrimSpace(meta.EntityName)
	if entityName == "" {
		entityName = "SARL INVESTISSIMO"
	}
	firm := strings.TrimSpace(meta.AccountantFirm)
	if firm == "" {
		firm = "Cabinet SGCAM — Société de Gestion et de Comptabilité"
	}
	fiscalYear := meta.FiscalYear
	if fiscalYear <= 0 {
		fiscalYear = 2024
	}
	periodStart := strings.TrimSpace(meta.PeriodStart)
	if periodStart == "" {
		periodStart = fmt.Sprintf("01/01/%d", fiscalYear)
	}
	periodEnd := strings.TrimSpace(meta.PeriodEnd)
	if periodEnd == "" {
		periodEnd = fmt.Sprintf("31/12/%d", fiscalYear)
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html><html lang=\"fr\"><head><meta charset=\"UTF-8\">")
	sb.WriteString(fmt.Sprintf("<title>Comptes Annuels &amp; Liasse Fiscale %d — %s</title>", fiscalYear, html.EscapeString(entityName)))
	sb.WriteString("<style>body{font-family:Arial,sans-serif;font-size:11px;color:#0f172a}table{width:100%;border-collapse:collapse;margin-bottom:16px}th,td{border:1px solid #cbd5e1;padding:4px 6px;text-align:left}th{background:#f1f5f9}.num{text-align:right;font-variant-numeric:tabular-nums}.section{page-break-before:always}</style>")
	sb.WriteString("</head><body>")

	// 1. Page de garde & Sommaire (Cabinet SGCAM layout)
	sb.WriteString(fmt.Sprintf("<section id=\"cover\"><h1>COMPTES ANNUELS — EXERCICE %d</h1>", fiscalYear))
	sb.WriteString(fmt.Sprintf("<h2>%s (%s)</h2>", html.EscapeString(entityName), html.EscapeString(meta.EntityType)))
	sb.WriteString(fmt.Sprintf("<p><strong>Adresse :</strong> %s | <strong>SIRET :</strong> %s | <strong>Code NAF :</strong> %s</p>",
		html.EscapeString(meta.Address), html.EscapeString(meta.SIRET), html.EscapeString(meta.NAFCode)))
	sb.WriteString(fmt.Sprintf("<p><strong>Période :</strong> du %s au %s | <strong>Cabinet :</strong> %s</p>",
		html.EscapeString(periodStart), html.EscapeString(periodEnd), html.EscapeString(firm)))
	sb.WriteString("<h3>SOMMAIRE</h3><ul>" +
		"<li>Attestation de Présentation des Comptes</li>" +
		"<li>Bilan Actif (CERFA N° 2050)</li>" +
		"<li>Bilan Passif (CERFA N° 2051)</li>" +
		"<li>Compte de Résultat de l'Exercice (CERFA N° 2052 / 2053)</li>" +
		"<li>Détail Bilan Actif, Passif et Compte de Résultat (Variations N / N-1)</li>" +
		"<li>Soldes Intermédiaires de Gestion (SIG)</li>" +
		"<li>Liasse Fiscale : CERFA 2065 / 2072, 2054 Immobilisations, 2055 Amortissements, 2057 Créances et Dettes, 2058-A/B/C Résultat Fiscal, 2059-F Capital Social, 1330-CVAE</li>" +
		"</ul></section>")

	// 2. CERFA 2050 — Bilan Actif & CERFA 2051 — Bilan Passif
	sb.WriteString("<section class=\"section\" id=\"cerfa-2050-2051\"><h2>CERFA N° 2050 &amp; 2051 — BILAN ACTIF ET PASSIF</h2>")
	sb.WriteString("<table><thead><tr><th>Rubrique Bilan</th><th class=\"num\">Net N</th><th class=\"num\">Net N-1</th></tr></thead><tbody>")
	sb.WriteString(fmt.Sprintf("<tr><td>Actif Immobilisé Net (2050)</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.FixedAssetsNetNCents.FormatEUR()),
		html.EscapeString(stmt.FixedAssetsNetNMinus1Cents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<tr><td>Actif Circulant Net (2050)</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.CurrentAssetsNetNCents.FormatEUR()),
		html.EscapeString(stmt.CurrentAssetsNetNMinus1Cents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<tr><td><strong>TOTAL ACTIF (2050)</strong></td><td class=\"num\"><strong>%s</strong></td><td class=\"num\"><strong>%s</strong></td></tr>",
		html.EscapeString(stmt.TotalAssetsNCents.FormatEUR()),
		html.EscapeString(stmt.TotalAssetsNMinus1Cents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<tr><td>Capitaux Propres hors Résultat (2051)</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.EquityExclNetIncomeNCents.FormatEUR()),
		html.EscapeString(stmt.EquityExclNetIncomeNMinus1Cents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<tr><td>Résultat de l'exercice (Compte 120/129 - 2051 DI)</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.BalanceSheetNetIncomeNCents.FormatEUR()),
		html.EscapeString(stmt.BalanceSheetNetIncomeNMinus1Cents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<tr><td>Total Capitaux Propres (2051 DL)</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.TotalEquityNCents.FormatEUR()),
		html.EscapeString(stmt.TotalEquityNMinus1Cents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<tr><td>Total Dettes (Emprunts, Cautionnements, Associés, Fournisseurs - 2051 EC)</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.TotalLiabilitiesNCents.FormatEUR()),
		html.EscapeString(stmt.TotalLiabilitiesNMinus1Cents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<tr><td><strong>TOTAL PASSIF (2051 EE)</strong></td><td class=\"num\"><strong>%s</strong></td><td class=\"num\"><strong>%s</strong></td></tr>",
		html.EscapeString(stmt.TotalEquityAndLiabilitiesNCents.FormatEUR()),
		html.EscapeString(stmt.TotalEquityAndLiabilitiesNMinus1Cents.FormatEUR())))
	sb.WriteString("</tbody></table></section>")

	// 3. CERFA 2052 / 2053 & Détail Compte de Résultat (only active lines!)
	sb.WriteString("<section class=\"section\" id=\"cerfa-2052-2053\"><h2>CERFA N° 2052 / 2053 — COMPTE DE RÉSULTAT &amp; DÉTAIL DES COMPTES</h2>")
	sb.WriteString("<table><thead><tr><th>Compte</th><th>Intitulé</th><th>Section</th><th class=\"num\">Exercice N</th><th class=\"num\">Exercice N-1</th><th class=\"num\">Variation €</th><th class=\"num\">Variation %</th></tr></thead><tbody>")
	for _, line := range stmt.Lines {
		if line.Disabled {
			continue
		}
		sb.WriteString(fmt.Sprintf(
			"<tr><td>%s</td><td>%s</td><td>%s</td><td class=\"num\">%s</td><td class=\"num\">%s</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
			html.EscapeString(line.AccountCode),
			html.EscapeString(line.Label),
			html.EscapeString(translateStatementSectionFr(line.Section)),
			html.EscapeString(line.NetNCents.FormatEUR()),
			html.EscapeString(line.NetNMinus1Cents.FormatEUR()),
			html.EscapeString(line.VariationAbsCents.FormatEUR()),
			html.EscapeString(domain.FormatVariationPercent(line.NetNCents, line.NetNMinus1Cents)),
		))
	}
	sb.WriteString(fmt.Sprintf("<tr><td colspan=\"3\"><strong>TOTAL PRODUITS D'EXPLOITATION (2052 FR)</strong></td><td class=\"num\"><strong>%s</strong></td><td class=\"num\"><strong>%s</strong></td><td></td><td class=\"num\"><strong>%s</strong></td></tr>",
		html.EscapeString(stmt.TotalOperatingRevenueNCents.FormatEUR()),
		html.EscapeString(stmt.TotalOperatingRevenueNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.TotalOperatingRevenueNCents, stmt.TotalOperatingRevenueNMinus1Cents))))
	sb.WriteString(fmt.Sprintf("<tr><td colspan=\"3\"><strong>TOTAL CHARGES D'EXPLOITATION (2052 GF)</strong></td><td class=\"num\"><strong>%s</strong></td><td class=\"num\"><strong>%s</strong></td><td></td><td class=\"num\"><strong>%s</strong></td></tr>",
		html.EscapeString(stmt.TotalOperatingExpensesNCents.FormatEUR()),
		html.EscapeString(stmt.TotalOperatingExpensesNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.TotalOperatingExpensesNCents, stmt.TotalOperatingExpensesNMinus1Cents))))
	sb.WriteString(fmt.Sprintf("<tr><td colspan=\"3\"><strong>RÉSULTAT D'EXPLOITATION (2052 GG)</strong></td><td class=\"num\"><strong>%s</strong></td><td class=\"num\"><strong>%s</strong></td><td></td><td class=\"num\"><strong>%s</strong></td></tr>",
		html.EscapeString(stmt.OperatingIncomeNCents.FormatEUR()),
		html.EscapeString(stmt.OperatingIncomeNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.OperatingIncomeNCents, stmt.OperatingIncomeNMinus1Cents))))
	sb.WriteString(fmt.Sprintf("<tr><td colspan=\"3\"><strong>BÉNÉFICE OU PERTE — RÉSULTAT DE L'EXERCICE (2053 HN)</strong></td><td class=\"num\"><strong>%s</strong></td><td class=\"num\"><strong>%s</strong></td><td></td><td class=\"num\"><strong>%s</strong></td></tr>",
		html.EscapeString(stmt.NetIncomeNCents.FormatEUR()),
		html.EscapeString(stmt.NetIncomeNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.NetIncomeNCents, stmt.NetIncomeNMinus1Cents))))
	sb.WriteString("</tbody></table></section>")

	// 4. Soldes Intermédiaires de Gestion (SIG)
	sb.WriteString("<section class=\"section\" id=\"sig\"><h2>SOLDES INTERMÉDIAIRES DE GESTION (SIG)</h2>")
	sb.WriteString("<table><thead><tr><th>Indicateur SIG</th><th class=\"num\">Exercice N</th><th class=\"num\">Exercice N-1</th><th class=\"num\">Variation %</th></tr></thead><tbody>")
	sb.WriteString(fmt.Sprintf("<tr><td>Production de l'exercice</td><td class=\"num\">%s</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.TotalOperatingRevenueNCents.FormatEUR()),
		html.EscapeString(stmt.TotalOperatingRevenueNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.TotalOperatingRevenueNCents, stmt.TotalOperatingRevenueNMinus1Cents))))
	sb.WriteString(fmt.Sprintf("<tr><td>Valeur Ajoutée produite</td><td class=\"num\">%s</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.ValueAddedNCents.FormatEUR()),
		html.EscapeString(stmt.ValueAddedNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.ValueAddedNCents, stmt.ValueAddedNMinus1Cents))))
	sb.WriteString(fmt.Sprintf("<tr><td>Excédent Brut d'Exploitation (EBE)</td><td class=\"num\">%s</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.GrossOperatingSurplusNCents.FormatEUR()),
		html.EscapeString(stmt.GrossOperatingSurplusNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.GrossOperatingSurplusNCents, stmt.GrossOperatingSurplusNMinus1Cents))))
	sb.WriteString(fmt.Sprintf("<tr><td>Résultat d'Exploitation</td><td class=\"num\">%s</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
		html.EscapeString(stmt.OperatingIncomeNCents.FormatEUR()),
		html.EscapeString(stmt.OperatingIncomeNMinus1Cents.FormatEUR()),
		html.EscapeString(domain.FormatVariationPercent(stmt.OperatingIncomeNCents, stmt.OperatingIncomeNMinus1Cents))))
	sb.WriteString(fmt.Sprintf("<tr><td>Capacité d'Autofinancement (CAF)</td><td class=\"num\">%s</td><td class=\"num\">%s</td><td class=\"num\">-</td></tr>",
		html.EscapeString(stmt.SelfFinancingCapacityNCents.FormatEUR()),
		html.EscapeString(stmt.SelfFinancingCapacityNMinus1Cents.FormatEUR())))
	sb.WriteString("</tbody></table></section>")

	// 5. CERFA 2065 / 2072, 2054, 2055, 2057, 2058-A/B/C, 2059-F, 1330-CVAE
	sb.WriteString("<section class=\"section\" id=\"liasse-cerfa\"><h2>LIASSE FISCALE — CERFA 2065, 2054, 2055, 2057, 2058-A/B/C, 2059-F, 1330-CVAE, 2072</h2>")
	sb.WriteString(fmt.Sprintf("<p><strong>CERFA N° 2065 / 2072 :</strong> Résultat fiscal déclaré : %s | <strong>CERFA N° 1330-CVAE :</strong> Valeur Ajoutée de référence : %s</p>",
		html.EscapeString(stmt.NetIncomeNCents.FormatEUR()),
		html.EscapeString(stmt.ValueAddedNCents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<p><strong>CERFA N° 2054 &amp; 2055 (Immobilisations &amp; Amortissements) :</strong> Brut = %s | Amortissements = %s | Net = %s</p>",
		html.EscapeString(stmt.FixedAssetsGrossNCents.FormatEUR()),
		html.EscapeString(stmt.FixedAssetsDepreciationNCents.FormatEUR()),
		html.EscapeString(stmt.FixedAssetsNetNCents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<p><strong>CERFA N° 2057 (État des Créances et des Dettes) :</strong> Créances &amp; disponibilités = %s | Dettes = %s</p>",
		html.EscapeString(stmt.CurrentAssetsNetNCents.FormatEUR()),
		html.EscapeString(stmt.TotalLiabilitiesNCents.FormatEUR())))
	sb.WriteString(fmt.Sprintf("<p><strong>CERFA N° 2058-A / 2058-B / 2058-C :</strong> Résultat comptable et fiscal = %s</p>",
		html.EscapeString(stmt.NetIncomeNCents.FormatEUR())))

	sb.WriteString(fmt.Sprintf("<h3>CERFA N° 2059-F — COMPOSITION DU CAPITAL SOCIAL (Capital : %s — %d parts)</h3>",
		html.EscapeString(meta.ShareCapitalCents.FormatEUR()), meta.TotalShares))
	if len(meta.Shareholders) > 0 {
		sb.WriteString("<table><thead><tr><th>Associé</th><th>Qualité</th><th class=\"num\">Parts</th><th class=\"num\">% Détention</th><th class=\"num\">Compte Courant (455)</th><th class=\"num\">Rémunération</th></tr></thead><tbody>")
		for _, sh := range meta.Shareholders {
			pctWhole := sh.DetentionBasisPoints / 100
			pctFrac := sh.DetentionBasisPoints % 100
			sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td class=\"num\">%d</td><td class=\"num\">%d,%02d %%</td><td class=\"num\">%s</td><td class=\"num\">%s</td></tr>",
				html.EscapeString(sh.FullName),
				html.EscapeString(sh.Role),
				sh.SharesCount,
				pctWhole,
				pctFrac,
				html.EscapeString(sh.CurrentAccountCents.FormatEUR()),
				html.EscapeString(sh.RemunerationPaidCents.FormatEUR()),
			))
		}
		sb.WriteString("</tbody></table>")
	}
	sb.WriteString("</section></body></html>")

	return sb.String(), nil
}

// RenderAnnualAccountsPDF renders the Comptes Annuels & Liasse Fiscale HTML and converts it to PDF via Gotenberg/PdfEngine.
func (r *AnnualAccountsPDFRenderer) RenderAnnualAccountsPDF(
	ctx context.Context,
	meta domain.AnnualAccountsMetadata,
	stmt domain.ComputedFinancialStatementsDTO,
) ([]byte, error) {
	htmlDoc, err := r.RenderAnnualAccountsHTML(meta, stmt)
	if err != nil {
		return nil, err
	}
	return r.pdfEngine.RenderHTMLToPDF(ctx, htmlDoc)
}

func translateStatementSectionFr(sec string) string {
	switch domain.StatementSection(sec) {
	case domain.SectionOperatingRevenue:
		return "Produits d'exploitation"
	case domain.SectionExternalExpenses:
		return "Charges externes"
	case domain.SectionTaxesAndDuties:
		return "Impôts et taxes"
	case domain.SectionPersonnelExpenses:
		return "Charges de personnel"
	case domain.SectionDepreciation:
		return "Dotations aux amortissements"
	case domain.SectionOtherExpenses:
		return "Autres charges"
	case domain.SectionFinancialIncome:
		return "Produits financiers"
	case domain.SectionFinancialExpenses:
		return "Charges financières"
	case domain.SectionExtraordinaryIncome:
		return "Produits exceptionnels"
	case domain.SectionExtraordinaryExpenses:
		return "Charges exceptionnelles"
	case domain.SectionIncomeTax:
		return "Impôts sur les bénéfices"
	case domain.SectionFixedAssets:
		return "Actif immobilisé"
	case domain.SectionCurrentAssets:
		return "Actif circulant"
	case domain.SectionEquity:
		return "Capitaux propres"
	case domain.SectionLiabilities:
		return "Dettes"
	default:
		return sec
	}
}
