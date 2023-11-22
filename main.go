package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/aiviaio/go-binance/v2"
)

type TickerService struct {
	ticker *binance.ListSymbolTickerService
	done   context.Context
}

func NewTicker(client *binance.Client, done context.Context) *TickerService {
	return &TickerService{
		ticker: client.NewListSymbolTickerService(),
		done:   done,
	}
}

func (t *TickerService) Worker(symbol string, priceCh chan<- map[string]string) {
	res, err := t.ticker.Symbol(symbol).Do(t.done)
	if err != nil || len(res) < 1 {
		fmt.Println("err:", err)
		fmt.Println("len:", len(res))
		return
	}

	select {
	case <-t.done.Done():
		return
	case priceCh <- map[string]string{symbol: res[0].LastPrice}:
	}
}

const symbolsNumber = 5

func main() {
	client := binance.NewClient("", "")
	doneCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info, err := client.NewExchangeInfoService().Do(doneCtx)

	if err != nil {
		fmt.Println(err)
		return
	}

	ticker := NewTicker(client, doneCtx)

	priceCh := make(chan map[string]string)
	wg := &sync.WaitGroup{}
	symbols := info.Symbols[:symbolsNumber]

	for _, symbol := range symbols {
		wg.Add(1)

		go func(symbol string) {
			defer wg.Done()
			ticker.Worker(symbol, priceCh)
		}(symbol.Symbol)
	}

	for i := 0; i < symbolsNumber; i++ {
		select {
		case price := <-priceCh:
			fmt.Println(price)
		case <-doneCtx.Done():
			break
		}
	}

	wg.Wait()
	cancel()
	close(priceCh)
}
