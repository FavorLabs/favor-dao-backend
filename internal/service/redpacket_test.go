package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"sync"
	"testing"

	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/json"
	"github.com/redis/go-redis/v9"
)

func sum(numbers []string) (total string) {
	tmp := new(big.Int)
	for _, number := range numbers {
		tmp.Add(tmp, convert.StrTo(number).MustBigInt())
	}
	return tmp.String()
}

func testRedPacket(t *testing.T, vFunc func(string, int64) []string) {
	tests := []struct {
		price   string
		numbers int64
	}{
		{"200000", 1},
		{"100000", 2},
		{"20000", 30},
		{"1000", 4},
		{"5000", 9},
		{"100", 20},
	}

	one := new(big.Int).SetInt64(100)

	for _, test := range tests {
		packets := vFunc(test.price, test.numbers)
		if got := sum(packets); got != test.price {
			p := convert.StrTo(test.price).MustBigInt()
			gated := convert.StrTo(got).MustBigInt()
			t.Errorf("%d amount into %d , total acmout ne. got %d, want %d", p.Mul(p, one).Int64(),
				test.numbers, gated.Mul(gated, one).Int64(), p.Mul(p, one).Mul(p, one).Int64())
		} else {
			fmt.Println("redpacket:", packets)
		}
	}
}

func TestRedPacketLucked(t *testing.T) {
	fmt.Println("lucked")
	testRedPacket(t, redpacketLucked)
}

func TestRedpacket(t *testing.T) {
	packets := redpacketLucked("10", 5)
	fmt.Printf("%s\n", packets)
}

func TestClaimRedpacket(t *testing.T) {
	// t.Skip("redpacket feat, need run server")

	redpacketID := "64634f5c4497b9aee6f9c70a"
	totalUser := 300
	isRand := false
	testUser := func() int {
		return rand.Intn(1000)
	}
	ctx := context.TODO()
	var claimTotal int64 = 0
	wg := sync.WaitGroup{}
	for i := 0; i < totalUser; i++ {
		var v = i
		if isRand {
			v = testUser()
		}
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			url := fmt.Sprintf("http://127.0.0.1:8010/v1/test/redpacket/%s?user=test%d", redpacketID, k)
			var resp struct {
				Code    int                  `json:"code"`
				Msg     string               `json:"msg"`
				Data    model.RedpacketClaim `json:"data,omitempty"`
				Details []string             `json:"details"`
			}
			err := request(ctx, "POST", url, nil, &resp)
			if err != nil {
				t.Logf("test%d err:%s msg:%s details:%s", k, err, resp.Msg, resp.Details)
				for _, v := range resp.Details {
					t.Logf("test%d, details:%s", k, v)
				}
				return
			}
			if resp.Code != 0 {
				t.Logf("test%d errcode:%d, msg:%s", k, resp.Code, resp.Msg)
				return
			}
			t.Logf("test%d claim: %s", k, resp.Data.Amount)
			a, _ := new(big.Int).SetString(resp.Data.Amount, 10)
			claimTotal += a.Int64()
			// }(i)
		}(v)
	}
	wg.Wait()
	t.Logf("total claim: %d", claimTotal)
}

func request(ctx context.Context, method, url string, body, respData interface{}) error {
	client := http.DefaultClient
	var reqBody io.Reader
	if body != nil {
		rawBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(rawBody)
	} else {
		reqBody = nil
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	// if resp.StatusCode != http.StatusOK {
	// 	return errors.New(resp.Status)
	// }
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(respBody, respData)
}

func TestRedis(t *testing.T) {
	t.Skip("must connect redis")
	rd := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	v, err := rd.Get(context.Background(), "a").Int64()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(v)
}
