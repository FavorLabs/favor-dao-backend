package service

import (
	"math/rand"
	"time"
)

var defaultAvatars = []string{
	"oHe7v-UcFwoTW0CGtm-Ll24Ec5-iCvQUmoLfqn2tCC",
	"wLwVGnl18iL3pKFog6uoFp1NyFCQUA8F3yqKk0W_6W",
	"K5PsynSe2LZHQvWyyTJe3K_dkuw5PMTeU8o_d5NbUG",
	"qmUYaBjnX4zhSTXCSzOCOE-X5UwhJVxtLiST7vL4iK",
	"aXwnJ7XvXTJpZKLlZOAb9_KPty2JyGHDT2ZifSE3h3",
	"TYYNtQriE7M9OIwt5JjybtoqOf3oCQTcIfAs4atL3-",
	"GnpxC_Y_TcvMzFJarnkbKekS5TGUUSxZA2D_4FWGQr",
	"t6dmma3pYBoFVdpSMLn-cxrqmOEPaMitdSULvem6D2",
	"v1WpwMRTAFqt4ltk_Spp_QGiWuydHrxgBWArQO9YWy",
	"UaNXsZM-LZ6Vedsrd2RmzajmpuGpANTx90vcn2kuPK",
	"PXELs_2TN6X0McRa4W7svsJQ0fi3kSV3xb9khhZ_2U",
	"IIKmKlrFrjnsjxyGNxHIkoryLNjdiR9VNWCpMVsj7M",
	"nZVjEuazBbEW0YFHx73-lYQBKhdnPLUw9qf34lzTQk",
	"36aMJzfxsdPk5ujjFI1pK9eG6xRWmPPMnAVDiy65uv",
	"B2Grv_tOCnfJfUJa2NTs8XoDCslkTvAmS4KRLRSUHL",
	"YcwboaV330Gv2qok_w43sGQLbtsXEyt3TEEcyBrrX5",
	"Sa1T9HpxMTwym14qiQIbppiKwmmoo_gqDr3heRpqVb",
	"92xw2Ng_RxcxERN5Ylc8JWu5jImBeQMWzbgusu-yux",
	"-RaQo55FVZBn5L8jrrqjz-RyarJcNKEhJZfAVs7amU",
	"7cM1oonpdTGJQS6y7Hj_6CLlbWEwGM5AFVqBkE9t_3",
	"S6ncjc467crO9sIleD044_kmhOgSwvXxy2_pexJiO9",
	"MjAlOmx75R3Snif2q-ShbT5D8_M7bjjlhv3741jx0E",
	"NNHgmYRCqmUhv2euYuUYTQZyumCmiLNz8SDtmw5UGQ",
	"-Jk_b9yiaojKpyTieo_cbdV9Rv6E2Vdd4tj0CTHeow",
	"c8zjr0ix8GPnrCzXgvRAZOrXx6SYpwp_Sx4yoGlwQ7",
	"rBNZfbvt_QJQGC0Fej7fDemvJHd6GU6O-IYvfX_So6",
	"3eR_-4WUrlsOjy4lZTJgrDIfCSKbON15EQkJocZFcC",
	"a5GD0mLHIo8xv-R2ObF_aO-xI52nEvzh6c-zDw5gdo",
	"5__JcSfnWqoXJGd2mS2LbP4CZXjPRH9AC056lXCe4q",
	"j33g9I4M_HdzbzORQyBi9_DrxMFgWeqR0g_TkWmkNw",
	"jQpfKLKTp6d3p-Ys1kjoimVRIqq93OV9Xt_LZZ-e-S",
	"Ntyjm5MpXGSzQMa2b8xsWWXo818VtekR3I-gdeysHl",
	"viHsCtBZfTNkGH40PbBxB3pzhqJUU7niQ1UVknaWtZ",
	"5Ealw6SNlJS5wOBJJQpJlnhIjg2wrARtgSmdR7dKn3",
	"-5zzX1KMxWNiwP6-hhAZsu6oskDVl4gMO1VJhcEM5a",
	"IDaNtq21ma5MrIsmd4O3LrZcbZnll2-Jvk8xJNcIvZ",
	"zIwGRxEyJolbl90Z2Fusqj-PGToYDvDYBOy1l6zKph",
	"34P_LnEI6a-L3VJXyuwRcFkRSz74HQ1wPcNvrnzwEj",
	"NtM1zu0rvyAHDaUQI6WGEqOLKb-uOiFGgKCguMBxDD",
	"6zrkxbayuHWCPO-gtWmBuRikyhXpMIkg2xgJG0VLFe",
	"XERf9RqYwszmAvlFbHnMEK7Q_aIRO_d_PeuV-0Rgx5",
	"Fvzp2rlEn7uKK-Kz1bDpjUswmfKnWfuKfMwVIdOpQL",
	"kKlRG-nFEqOOqpkLsPEsOFWTpcZ4UFJL9EO-QZP8_n",
	"rEqYhx9d4RJDMOGoI7IQjuUYlmtOmqPXJ_Adh03XvN",
	"L0rjM66cqBBultZ4uaGJblOdpWm5cvgba5HEU46B8q",
	"djUdMTvrht2GKDPMBK55DfRTan_vLIuNAHwOdOdQ2X",
	"8ITTus_PMFPs3LCNK56GHsTr3ERO3ZgQf-APTnbrWI",
	"9ZQ5eIAGIesDqNnKGyhLS7ra-YRvBh0GcCvijBp1gN",
	"Jw0eVRKsCYrwfc2dYtLxivNNaHx9Fj0vp46_Xob8Y8",
	"onMEcdZyfFe-Sth9EQpMsGZ9s3GfPf6owDn1rvHqv8",
}

func GetRandomAvatar() string {
	rand.Seed(time.Now().UnixMicro())
	return defaultAvatars[rand.Intn(len(defaultAvatars))]
}
