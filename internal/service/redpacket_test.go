package service

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"github.com/go-redis/redis/v8"
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
	packets := redpacketLucked("0xa", 5)
	fmt.Printf("%s\n", packets)
}

func TestClaimRedpacket(t *testing.T) {
	t.Skip("must connect redis")
	// test rpush/rpop
	rd := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	if err := rd.Ping(context.TODO()).Err(); err != nil {
		t.Fatalf("new redis failed: %s", err)
	}

	ctx := context.TODO()
	packets := redpacketLucked("10", 3)
	err := rd.RPush(ctx, "test1", packets).Err()
	if err != nil {
		t.Fatal(err)
	}
	i, err := rd.RPop(ctx, "no_key").Int64()
	if err != nil {
		// redis nil
		t.Log(err)
	}
	if i != 0 {
		t.Fatalf("expected 0, got i")
	}

	i, err = rd.RPop(ctx, "test1").Int64()
	if err != nil {
		t.Fatal(err)
	}
	if i <= 0 {
		t.Fatalf("expected x, got <=0")
	}
}

func TestCreateRedpacket(t *testing.T) {
	conf.Initialize([]string{}, false, "../../")
	Initialize()
	rid, err := CreateRedpacket("0x1717fa888b3392db23258ad46298ff75b597a060", RedpacketRequest{
		Type:   model.RedpacketTypeLucked,
		Title:  "test lucked",
		Amount: "103",
		Total:  2,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("redpacket_id: %s", rid)
}
