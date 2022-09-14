package triggers

import "time"

type aggregator struct {
	ticker time.Ticker
}

func newAggregator(ticker time.Ticker) *aggregator {
	return &aggregator{ticker: ticker}
}

func (a *aggregator) Run() {

}
