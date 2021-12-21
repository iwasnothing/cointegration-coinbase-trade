### Trade CyptoCurrency using Cointegration
# Trade Strategy
- [My Blog]

# Execution
1. export the following Environment Variables for the Coinbase Pro API credentials, and Trading parameters.
```
export S1=ETH
export S2=BTC
export Key=
export Passphrase=
export Secret=
export Intercept=
export Beta=
export Lookback=5
```
2.  compile the source by 
```
go build .
```
3.  run the executable for each day.  
```
./cointRealTrade
```
