package conf

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestUseDefault(t *testing.T) {
	suites := map[string][]string{
		"default": {"Sms", "Alipay", "Zinc", "MySQL", "Redis", "AliOSS", "LogZinc"},
		"develop": {"Zinc", "MySQL", "AliOSS", "LogFile"},
		"slim":    {"Zinc", "MySQL", "Redis", "AliOSS", "LogFile"},
	}
	kv := map[string]string{
		"sms": "SmsJuhe",
	}
	features := newFeatures(suites, kv)
	for _, data := range []struct {
		key    string
		expect string
		exist  bool
	}{
		{"Sms", "SmsJuhe", true},
		{"Alipay", "", true},
		{"Zinc", "", true},
		{"Redis", "", true},
		{"Database", "", false},
	} {
		if v, ok := features.Cfg(data.key); ok != data.exist || v != data.expect {
			t.Errorf("key: %s expect: %s exist: %t got v: %s ok: %t", data.key, data.expect, data.exist, v, ok)
		}
	}
	for exp, res := range map[string]bool{
		"Sms":           true,
		"Sms = SmsJuhe": true,
		"SmsJuhe":       false,
		"default":       true,
	} {
		if ok := features.CfgIf(exp); res != ok {
			t.Errorf("CfgIf(%s) want %t got %t", exp, res, ok)
		}
	}
}

func TestUse(t *testing.T) {
	suites := map[string][]string{
		"default": {"Sms", "Alipay", "Zinc", "MySQL", "Redis", "AliOSS", "LogZinc"},
		"develop": {"Zinc", "MySQL", "AliOSS", "LogFile"},
		"slim":    {"Zinc", "MySQL", "Redis", "AliOSS", "LogFile"},
	}
	kv := map[string]string{
		"sms": "SmsJuhe",
	}
	features := newFeatures(suites, kv)

	features.Use([]string{"develop"}, true)
	for _, data := range []struct {
		key    string
		expect string
		exist  bool
	}{
		{"Sms", "", false},
		{"Alipay", "", false},
		{"Zinc", "", true},
		{"Redis", "", false},
		{"Database", "", false},
	} {
		if v, ok := features.Cfg(data.key); ok != data.exist || v != data.expect {
			t.Errorf("key: %s expect: %s exist: %t got v: %s ok: %t", data.key, data.expect, data.exist, v, ok)
		}
	}
	for exp, res := range map[string]bool{
		"Sms":           false,
		"Sms = SmsJuhe": false,
		"SmsJuhe":       false,
		"default":       false,
		"develop":       true,
	} {
		if ok := features.CfgIf(exp); res != ok {
			t.Errorf("CfgIf(%s) want %t got %t", exp, res, ok)
		}
	}

	features.UseDefault()
	features.Use([]string{"slim", "", "demo"}, false)
	for _, data := range []struct {
		key    string
		expect string
		exist  bool
	}{
		{"Sms", "SmsJuhe", true},
		{"Alipay", "", true},
		{"Zinc", "", true},
		{"Redis", "", true},
		{"Database", "", false},
		{"demo", "", true},
	} {
		if v, ok := features.Cfg(data.key); ok != data.exist || v != data.expect {
			t.Errorf("key: %s expect: %s exist: %t got v: %s ok: %t", data.key, data.expect, data.exist, v, ok)
		}
	}
	for exp, res := range map[string]bool{
		"Sms":           true,
		"Sms = SmsJuhe": true,
		"SmsJuhe":       false,
		"default":       true,
		"develop":       false,
		"slim":          true,
		"demo":          true,
	} {
		if ok := features.CfgIf(exp); res != ok {
			t.Errorf("CfgIf(%s) want %t got %t", exp, res, ok)
		}
	}
}

func TestCfgIf(t *testing.T) {
	url := "https://235461af0053efb9.apiclient-us.cometchat.io/v3/groups/1/members"

	payload := strings.NewReader("{\"participants\":[\"0x123\"]}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	// req.Header.Add("apikey", "e50156f5ab1294f3f1f67bf685d1a08e9245eea7")
	req.Header.Add("appid", "235461af0053efb9")
	// req.Header.Add("apikey", "0x123_1679885131ab2597f14afcae5319c22c89402ecf")
	req.Header.Add("authToken", "0x123_1679885131ab2597f14afcae5319c22c89402ecf")
	// req.Header.Add("resource", "false")
	// req.Header.Add("sdk", "android@3.0.12")

	res, _ := http.DefaultClient.Do(req)

	defer func() {
		_ = res.Body.Close()
	}()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(res)
	fmt.Println(string(body))
}
