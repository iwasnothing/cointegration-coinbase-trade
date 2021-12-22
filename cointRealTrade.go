package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	coinbasepro "github.com/preichenberger/go-coinbasepro/v2"
	"gonum.org/v1/gonum/stat"
)

type AcStatus struct {
	product  string
	balance  float64
	position int64
	lastbuy  float64
}

func PctChg(x []float64) []float64 {
	var pct []float64
	for i, v := range x {
		if i > 0 {
			pct = append(pct, (v-x[i-1])/x[i-1])
		}
	}
	return pct
}
func createCoinClient() *coinbasepro.Client {
	client := coinbasepro.NewClient()
	client.UpdateConfig(&coinbasepro.ClientConfig{
		BaseURL:    "https://api.pro.coinbase.com",
		Key:        os.Getenv("Key"),
		Passphrase: os.Getenv("Passphrase"),
		Secret:     os.Getenv("Secret"),
	})

	return client
}
func getData(s1 string, lookback int, client *coinbasepro.Client) ([]float64, []float64) {
	to := time.Now()
	//fmt.Println(s1)
	//fmt.Println(to)
	from := to.AddDate(0, 0, 0-lookback)
	fmt.Println(from, to)
	rateParm := coinbasepro.GetHistoricRatesParams{
		Start:       from,
		End:         to,
		Granularity: 86400,
	}
	var n int
	var rates []coinbasepro.HistoricRate
	var err error
	for ok := true; ok; ok = (err != nil || n == 0) {
		rates, err = client.GetHistoricRates(s1+"-USDT", rateParm)
		if err != nil {
			println(err.Error())
		} else {
			n = len(rates)
			fmt.Println("get rate", n)
		}
	}
	var p1 []float64
	for _, t := range rates {
		p1 = append(p1, t.Close)
		//fmt.Println(t.Time)
	}
	//fmt.Println("first rate", rates[0].Time)
	pct1 := PctChg(p1)

	return p1, pct1
}
func getCurrent(s1 string, client *coinbasepro.Client) (float64, float64, float64) {

	tick, err := client.GetTicker(s1)
	if err != nil {
		println(err.Error())
	}
	fmt.Println("getcurrent", tick.Price, tick.Bid, tick.Ask, tick.Time.Time().String())
	price, _ := strconv.ParseFloat(tick.Price, 64)
	ask, _ := strconv.ParseFloat(tick.Ask, 64)
	bid, _ := strconv.ParseFloat(tick.Bid, 64)
	return price, ask, bid
}
func getSignal(s1 string, s2 string, lookback int, beta float64, res float64) (int, int) {
	c := createCoinClient()
	p1, pct1 := getData(s1, lookback, c)
	//fmt.Println(p1, pct1)
	p2, pct2 := getData(s2, lookback, c)
	//fmt.Println(p2, pct2)
	//N := len(p1)
	//n := len(pct1)
	var residues []float64
	for i, v1 := range pct1 {
		intercept := v1 - beta*pct2[i]
		residues = append(residues, intercept)
		//fmt.Println(len(residues))

	}
	std12 := stat.StdDev(residues[1:], nil)
	fmt.Println("std residual", std12)

	current1, bid1, ask1 := getCurrent(s1+"-USDT", c)
	rtn1 := (current1 - p1[1]) / p1[1]
	fmt.Println("return1", rtn1, current1, p1[0], bid1, ask1)

	current2, bid2, ask2 := getCurrent(s2+"-USDT", c)
	rtn2 := (current2 - p2[1]) / p2[1]
	fmt.Println("return2", rtn2, current2, p2[0], bid2, ask2)
	dif12 := rtn1 - beta*rtn2
	fmt.Println("dif12", dif12)
	half := int(lookback / 2)
	fmt.Println("half lookback", half)

	mm1 := stat.Mean(p1[:half], nil) - stat.Mean(p1, nil)
	mm2 := stat.Mean(p2[:half], nil) - stat.Mean(p2, nil)
	fmt.Println("mm", mm1, mm2)
	var sig1 = 0
	var sig2 = 0
	var thd1 = 2.0

	if dif12 > res+thd1*std12 {
		sig1 = -1
		sig2 = 1
	} else if dif12 < res-thd1*std12 {
		sig1 = 1
		sig2 = -1
	}

	return sig1, sig2

}
func countOrder(s1 string) int {
	client := createCoinClient()
	var orders []coinbasepro.Order
	var orderParm coinbasepro.ListOrdersParams
	orderParm.Status = "open"
	orderParm.ProductID = s1
	total := 0
	cursor := client.ListOrders(orderParm)
	//fmt.Println("countOrder")
	for cursor.HasMore {
		if err := cursor.NextPage(&orders); err != nil {
			fmt.Println(err.Error())
			return 0
		}

		fmt.Println("has orders", cursor.Pagination.Limit)

		for _, o := range orders {
			fmt.Println("1 order", o.ID, o.Status, o.ProductID)
			total = total + 1
		}

	}
	return total
}
func placeOrder(product string, buysell string, price float64, amount float64) {
	order := coinbasepro.Order{
		Price:     fmt.Sprintf("%f", price),
		Size:      fmt.Sprintf("%.6f", amount),
		Side:      buysell,
		ProductID: product,
	}
	fmt.Println("create order", order)
	client := createCoinClient()
	savedOrder, err := client.CreateOrder(&order)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("placed order", savedOrder)
}
func execOrder(s1 string, s2 string, sig1 int, sig2 int) {
	c := createCoinClient()
	currentStatus1 := readStatus(s1)
	currentStatus2 := readStatus(s2)
	product := s1 + "-" + s2
	current1, bid1, ask1 := getCurrent(product, c)

	cnt := countOrder(product)
	fmt.Println("count order", cnt)
	minus_fee := 0.99
	fmt.Println("current "+product, current1, bid1, ask1)
	if currentStatus2.balance > 0 && sig1 == 1 && sig2 == -1 {
		fmt.Println("buy "+product, current1, currentStatus2.balance/current1*minus_fee)
		placeOrder(product, "buy", current1, currentStatus2.balance/current1*minus_fee)
	} else if currentStatus1.balance > 0 && sig1 == -1 && sig2 == 1 {
		fmt.Println("sell "+product, current1, currentStatus1.balance/current1*minus_fee)
		placeOrder(product, "sell", current1, currentStatus1.balance/current1*minus_fee)
	} else {
		fmt.Println("no action")
	}

}

func readStatus(s1 string) AcStatus {
	var initial AcStatus
	client := createCoinClient()
	accounts, err := client.GetAccounts()
	if err != nil {
		fmt.Println(err.Error())
	}
	initial.product = s1
	initial.balance = 0
	initial.position = 0
	for _, a := range accounts {

		if s1 == a.Currency {
			initial.product = a.Currency

			bal, _ := strconv.ParseFloat(a.Balance, 64)
			initial.balance = bal
			println(a.Currency, " balance status", bal)
			if bal > 0 {
				initial.position = 1
			} else {
				initial.position = 0
			}

		}
	}
	fmt.Println("initial status", initial)
	return initial
}

func main() {
	var sig1 = 0
	var sig2 = 0
	var s1 = os.Getenv("S1")
	var s2 = os.Getenv("S2")
	fmt.Println(s1, s2)
	//mystate1 := readStatus(s1)
	//mystate2 := readStatus(s2)
	//fmt.Println(mystate1, mystate2)
	//var lookback = 21
	//var beta = 0.63900213802213
	//var intercept = 0.00594209736694315
	var intercept, _ = strconv.ParseFloat(os.Getenv("Intercept"), 64)
	var beta, _ = strconv.ParseFloat(os.Getenv("Beta"), 64)
	var lookback, _ = strconv.Atoi(os.Getenv("Lookback"))
	fmt.Println("parameters", intercept, beta, lookback)
	sig1, sig2 = getSignal(s1, s2, lookback, beta, intercept)
	fmt.Println("today signal", sig1, sig2)
	execOrder(s1, s2, sig1, sig2)
	product := s1 + "-" + s2
	countOrder(product)
}
