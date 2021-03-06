/**
 * Copyright 2015 @ z3q.net.
 * name : partner_cache
 * author : jarryliu
 * date : -- :
 * description :
 * history :
 */
package cache

import (
	"bytes"
	"fmt"
	"github.com/jsix/gof/storage"
	"go2o/core/domain/interface/express"
	"go2o/core/domain/interface/merchant"
	"go2o/core/service/dps"
	"sort"
	"strconv"
	"strings"
)

// 获取商户信息缓存
func GetValueMerchantCache(merchantId int) *merchant.Merchant {
	var v merchant.Merchant
	var sto storage.Interface = GetKVS()
	var key string = GetValueMerchantCacheCK(merchantId)
	if sto.Get(key, &v) != nil {
		v2, err := dps.MerchantService.GetMerchant(merchantId)
		if v2 != nil && err == nil {
			sto.SetExpire(key, *v2, DefaultMaxSeconds)
			return v2
		}
	}
	return &v

}

// 设置商户信息缓存
func GetValueMerchantCacheCK(merchantId int) string {
	return fmt.Sprintf("cache:partner:value:%d", merchantId)
}

// 设置商户站点配置
func GetMerchantSiteConfCK(merchantId int) string {
	return fmt.Sprintf("cache:partner:siteconf:%d", merchantId)
}

func DelMerchantCache(merchantId int) {
	kvs := GetKVS()
	kvs.Del(GetValueMerchantCacheCK(merchantId))
	kvs.Del(GetMerchantSiteConfCK(merchantId))
}

// 根据主机头识别会员编号
func GetMerchantIdByHost(host string) int {
	merchantId := 0
	key := "cache:host-for:" + host
	sto := GetKVS()
	var err error
	if merchantId, err = sto.GetInt(key); err != nil || merchantId <= 0 {
		merchantId = dps.MerchantService.GetMerchantIdByHost(host)
		if merchantId > 0 {
			sto.SetExpire(key, merchantId, DefaultMaxSeconds)
		}
	}
	return merchantId
}

// 根据API ID获取商户ID
func GetMerchantIdByApiId(apiId string) int {
	var merchantId int
	kvs := GetKVS()
	key := fmt.Sprintf("cache:partner:api:id-%s", apiId)
	kvs.Get(key, &merchantId)
	if merchantId == 0 {
		merchantId = dps.MerchantService.GetMerchantIdByApiId(apiId)
		if merchantId != 0 {
			kvs.Set(key, merchantId)
		}
	}
	return merchantId
}

// 获取API 信息
func GetMerchantApiInfo(merchantId int) *merchant.ApiInfo {
	var d *merchant.ApiInfo = new(merchant.ApiInfo)
	kvs := GetKVS()
	key := fmt.Sprintf("cache:partner:api:info-%d", merchantId)
	err := kvs.Get(key, &d)
	if err != nil {
		if d = dps.MerchantService.GetApiInfo(merchantId); d != nil {
			kvs.Set(key, d)
		}
	}
	return d
}

var (
	expressCacheKey = "go2o:rep:express:ship-tab"
)

// 获取发货的快递选项卡
func GetShipExpressTab() string {
	sto := GetKVS()
	html, err := sto.GetString(expressCacheKey)
	if err != nil {
		html = getRealShipExpressTab()
		sto.Set(expressCacheKey, html)
	}
	return html
}

func getRealShipExpressTab() string {
	buf := bytes.NewBuffer(nil)
	list := dps.ExpressService.GetEnabledProviders()
	iMap := make(map[string][]*express.ExpressProvider, 0)
	letArr := []string{}
	for _, v := range list {
		for _, g := range strings.Split(v.GroupFlag, ",") {
			if g == "" {
				continue
			}
			arr, ok := iMap[g]
			if !ok {
				arr = []*express.ExpressProvider{}
				letArr = append(letArr, g)
			}
			arr = append(arr, v)
			iMap[g] = arr
		}

	}
	sort.Strings(letArr)
	l := len(letArr)
	if letArr[l-1] == "常用" {
		letArr = append(letArr[l-1:], letArr[:l-1]...)
	}

	buf.WriteString(`<div class="ui-tabs" id="tab_express"><ul class="tabs">`)
	for _, v := range letArr {
		buf.WriteString(`<li title="`)
		buf.WriteString(v)
		buf.WriteString(`" href="`)
		buf.WriteString(v)
		buf.WriteString(`"><span class="tab-title">`)
		buf.WriteString(v)
		buf.WriteString(`</li>`)
	}
	buf.WriteString("</ul>")
	buf.WriteString(`<div class="frames">`)
	i := 0
	for _, l := range letArr {
		buf.WriteString(`<div class="frame"><ul class="list">`)
		for _, v := range iMap[l] {
			i++
			buf.WriteString("<li><input type=\"radio\" name=\"ProviderId\" field=\"ProviderId\" value=\"")
			buf.WriteString(strconv.Itoa(v.Id))
			buf.WriteString(`" id="provider_`)
			buf.WriteString(strconv.Itoa(i))
			buf.WriteString(`"/><label for="provider_`)
			buf.WriteString(strconv.Itoa(i))
			buf.WriteString(`">`)
			buf.WriteString(v.Name)
			buf.WriteString("</label></li>")
		}
		buf.WriteString("</ul></div>")
	}
	buf.WriteString("</div></div>")
	return buf.String()
}
