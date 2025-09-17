package models

type Currency string

const (
	CurrencyRUB Currency = "RUB"
)

type VatCode int

const (
	VatCodeNoVat     VatCode = iota + 1 // Без НДС
	VatCode0Percent                     // НДС по ставке 0%
	VatCode10Percent                    // НДС по ставке 10%
	VatCode20Percent                    // НДС по ставке 20% (самый распространенный)
	VatCode10_110                       // Расчетный НДС 10/110
	VatCode20_120                       // Расчетный НДС 20/120
)
