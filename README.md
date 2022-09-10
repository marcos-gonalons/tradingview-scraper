# tradingview-scraper
Get any market data in real time from the TradingView socket :) Ready to use in your Golang projects!


Inspired by https://github.com/imxeno/tradingview-scraper, I decided to create my own implementation of the TradingView socket using Go


## Installation
```
go get github.com/marcos-gonalons/tradingview-scraper/v2@latest
```

## How to use
Call the Connect() function passing 2 callback functions; one callback for when new market data is read from the socket, and another one used if an error happens while the connection is active

```golang
import socket "github.com/marcos-gonalons/tradingview-scraper/v2"

func main() {
    tradingviewsocket, err := socket.Connect(
        func(symbol string, data *socket.QuoteData) {
            fmt.Printf("%#v", symbol)
            fmt.Printf("%#v", data)
        },
        func(err error, context string) {
            fmt.Printf("%#v", "error -> "+err.Error())
            fmt.Printf("%#v", "context -> "+context)
        },
    )
    if err != nil {
        panic("Error while initializing the trading view socket -> " + err.Error())
    }
}
```


## How to add / remove symbols
The implementation allows you to listen for any market data changes, in real time, for any market available in TradingView.
In order to tell the socket the symbols (markets) that we want to get the data from, we need to call socket.AddSymbol(), after the connection is stablished.
```golang
tradingviewsocket.AddSymbol("OANDA:EURUSD")
tradingviewsocket.AddSymbol("BITSTAMP:BTCUSD")
// etc etc
```
The syntax for the symbol needs to be `broker or exchange name`:`market`.
Everytime the socket receives new data from those markets, it will call your callback function.

If you want to stop receiving updates from a particular market, just call RemoveSymbol()
```golang
   tradingviewsocket.RemoveSymbol("OANDA:EURUSD")
```


## Callback function
The callback function has 2 parameters; the symbol (market) name, and the data.
The data is a struct with these parameters: `Price`, `Volume`, `Bid`, `Ask`
```golang
callbackFn := func(symbol string, data *socket.QuoteData) {
    fmt.Printf("%#v", symbol)
    fmt.Printf("%#v", data)
    if data.Price != nil {
        fmt.Printf("%#v", "Price has changed")
    }
    if data.Volume != nil {
        fmt.Printf("%#v", "Volume has changed")
    }
    if data.Bid != nil {
        fmt.Printf("%#v", "Bid has changed")
    }
    if data.Ask != nil {
        fmt.Printf("%#v", "Ask has changed")
    }
}
```
Everytime new data is received from the socket, it will call your callback function.
This means that not always all the parameters will be available; sometimes, only the bid changes, or only the price changes, or only the volume, or a combination of any of those. The ones that did not change will be `nil`, since all of them are pointers to float64.

### Buy me a coffee?
If you found this repository useful for your needs, please consider sending a donation :) I highly appreciate it
- Paypal: https://www.paypal.com/paypalme/mgonalonscamps
- Bitcoin: 3Ets7kos4fG4aiEmrBEmKKpKogPyJJsk9A
- Ethereum: 0x4E73fcd8847bf456789b5D34Ed483C0474426271
- ADA: addr1q8x0482rvp7wzjsladlfs89z5rg5pe4s8effvxan66ap8vnrl8psznmjw6cawc6qp89mp98fl4p63eknl8y545wdh4xqpjl08w
