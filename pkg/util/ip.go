package util

import "favor-dao-backend/pkg/util/iploc"

func GetIPLoc(ip string) string {
	country, _ := iploc.Find(ip)
	return country
}
