package currency

import (
	"github.com/shopspring/decimal"
)

type Currency struct {
	Base int64
}

func New(base int64) *Currency {
	return &Currency{Base: base}
}

func (c *Currency) ScaleDown(amount int64) float64 {
	amountInfo, _ := decimal.NewFromInt(amount).Div(decimal.NewFromInt(c.Base)).Round(2).Float64()
	return amountInfo
}

func (c *Currency) ScaleDownInt64(amount int64) int64 {
	amountInfo := decimal.NewFromInt(amount).Div(decimal.NewFromInt(c.Base)).Round(2).IntPart()
	return amountInfo
}

func (c *Currency) ScaleUp(amount int64) int64 {
	return amount * c.Base
}
