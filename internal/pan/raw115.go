package pan

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"panplayer/internal/config"

	driver "github.com/jianxcao/115driver/pkg/driver"
)

const (
	apiOpenUserInfo = "https://proapi.115.com/open/user/info"
	apiIndexInfo    = "https://webapi.115.com/files/index_info"
)

func loginWithQRCode(client *driver.Pan115Client, session *driver.QRCodeSession) (*config.Credential, error) {
	resp, err := client.NewRequest().
		SetFormData(map[string]string{
			"account": session.UID,
			"app":     string(driver.LoginAppWeb),
		}).
		ForceContentType("application/json;charset=UTF-8").
		Post(fmt.Sprintf(driver.ApiQrcodeLoginWithApp, driver.LoginAppWeb))
	if err != nil {
		return nil, err
	}

	body, err := decodeObject(resp.Body())
	if err != nil {
		return nil, errors.New("115 登录返回了无法识别的 JSON 数据")
	}

	if asInt64(body["state"]) != 1 {
		return nil, apiError(body, "115 登录失败")
	}

	data := asMap(body["data"])
	if credential := credentialFromAny(data["cookie"]); credential != nil {
		return credential, nil
	}
	if credential := credentialFromResponseCookies(resp.RawResponse); credential != nil {
		return credential, nil
	}

	return nil, errors.New("115 登录成功，但没有返回可用的 cookie")
}

func fetchUserView(client *driver.Pan115Client) (*UserView, error) {
	resp, err := client.NewRequest().
		SetQueryParam("_", strconv.FormatInt(time.Now().UnixMilli(), 10)).
		Get(driver.ApiUserInfo)
	if err != nil {
		return nil, err
	}

	body, err := decodeObject(resp.Body())
	if err != nil {
		return nil, errors.New("115 用户信息返回了无法识别的 JSON 数据")
	}

	if !isSuccessBody(body) {
		return nil, apiError(body, "获取 115 用户信息失败")
	}

	data := asMap(body["data"])
	view := &UserView{}
	mergeUserViewFromNav(view, data)

	if extra, extraErr := fetchOpenUserInfoData(client); extraErr == nil {
		mergeUserViewFromOpenProfile(view, extra)
	}
	if indexInfo, indexErr := fetchIndexInfoData(client); indexErr == nil {
		mergeUserViewFromIndexInfo(view, indexInfo)
	}

	if view.UserName == "" {
		view.UserName = "115 用户"
	}
	if view.VIPLabel == "" {
		if view.IsVIP {
			view.VIPLabel = "VIP"
		} else {
			view.VIPLabel = "普通用户"
		}
	}
	if view.SpaceTotal > 0 && view.SpaceRemain <= 0 && view.SpaceUsed > 0 && view.SpaceTotal >= view.SpaceUsed {
		view.SpaceRemain = view.SpaceTotal - view.SpaceUsed
	}
	return view, nil
}

func fetchOpenUserInfoData(client *driver.Pan115Client) (map[string]any, error) {
	resp, err := client.NewRequest().
		SetQueryParam("_", strconv.FormatInt(time.Now().UnixMilli(), 10)).
		Get(apiOpenUserInfo)
	if err != nil {
		return nil, err
	}

	body, err := decodeObject(resp.Body())
	if err != nil {
		return nil, err
	}

	success := isSuccessBody(body) || asInt64(body["code"]) == 0
	if !success {
		return nil, apiError(body, "获取 115 扩展用户信息失败")
	}
	return asMap(body["data"]), nil
}

func fetchIndexInfoData(client *driver.Pan115Client) (map[string]any, error) {
	resp, err := client.NewRequest().
		SetQueryParam("count_space_nums", "1").
		SetQueryParam("_", strconv.FormatInt(time.Now().UnixMilli(), 10)).
		Get(apiIndexInfo)
	if err != nil {
		return nil, err
	}

	body, err := decodeObject(resp.Body())
	if err != nil {
		return nil, err
	}

	success := isSuccessBody(body) || asInt64(body["code"]) == 0
	if !success {
		return nil, apiError(body, "获取 115 空间信息失败")
	}
	return asMap(body["data"]), nil
}

func mergeUserViewFromNav(view *UserView, data map[string]any) {
	if view == nil {
		return
	}

	if userID := asInt64(data["user_id"]); userID > 0 {
		view.UserID = userID
	}
	if userName := strings.TrimSpace(asString(data["user_name"])); userName != "" {
		view.UserName = userName
	}
	if faceURL := strings.TrimSpace(extractFaceURL(data["face"])); faceURL != "" {
		view.FaceURL = faceURL
	}
	if asInt64(data["vip"]) > 0 {
		view.IsVIP = true
	}
	if isTruthy(data["forever"]) {
		view.VIPForever = true
	}
	if expireAt := formatUnixishTime(firstPositiveInt64(data["expire"], data["expire_time"])); expireAt != "" {
		view.VIPExpireAt = expireAt
	}
}

func mergeUserViewFromOpenProfile(view *UserView, data map[string]any) {
	if view == nil {
		return
	}

	userInfo := asMap(firstNonNil(
		data["user_info"],
		data["userInfo"],
		data["profile"],
		data["user"],
	))
	if userID := asInt64(firstNonNil(userInfo["user_id"], data["user_id"])); userID > 0 {
		view.UserID = userID
	}
	if userName := strings.TrimSpace(firstNonEmpty(
		asString(userInfo["user_name"]),
		asString(data["user_name"]),
		asString(data["nickname"]),
	)); userName != "" {
		view.UserName = userName
	}
	if faceURL := strings.TrimSpace(firstNonEmpty(
		extractFaceURL(userInfo["face"]),
		extractFaceURL(data["face"]),
	)); faceURL != "" {
		view.FaceURL = faceURL
	}

	vipInfo := asMap(firstNonNil(
		data["vip_info"],
		data["vipInfo"],
	))
	if label := strings.TrimSpace(firstNonEmpty(
		asString(vipInfo["level_name"]),
		asString(vipInfo["vip_name"]),
		asString(vipInfo["name"]),
	)); label != "" {
		view.VIPLabel = label
	}
	if isTruthy(firstNonNil(vipInfo["is_vip"], vipInfo["vip"], data["vip"])) || view.VIPLabel != "" {
		view.IsVIP = true
	}
	if isTruthy(firstNonNil(vipInfo["forever"], vipInfo["is_forever"], data["forever"])) {
		view.VIPForever = true
	}
	if expireAt := formatUnixishTime(firstPositiveInt64(
		vipInfo["expire_time"],
		vipInfo["expire"],
		data["expire_time"],
		data["expire"],
	)); expireAt != "" {
		view.VIPExpireAt = expireAt
	}

	mergeUserViewSpaceInfo(view, asMap(firstNonNil(
		data["rt_space_info"],
		data["space_info"],
		data["spaceInfo"],
	)))
}

func mergeUserViewFromIndexInfo(view *UserView, data map[string]any) {
	if view == nil {
		return
	}
	mergeUserViewSpaceInfo(view, asMap(firstNonNil(
		data["space_info"],
		data["spaceInfo"],
	)))
}

func mergeUserViewSpaceInfo(view *UserView, spaceInfo map[string]any) {
	if view == nil {
		return
	}

	total := firstPositiveInt64(
		spaceMetricValue(spaceInfo["all_total"]),
		spaceMetricValue(spaceInfo["total"]),
		spaceMetricValue(spaceInfo["total_size"]),
	)
	used := firstPositiveInt64(
		spaceMetricValue(spaceInfo["all_use"]),
		spaceMetricValue(spaceInfo["used"]),
		spaceMetricValue(spaceInfo["use"]),
		spaceMetricValue(spaceInfo["used_size"]),
	)
	remain := firstPositiveInt64(
		spaceMetricValue(spaceInfo["all_remain"]),
		spaceMetricValue(spaceInfo["remain"]),
		spaceMetricValue(spaceInfo["free"]),
		spaceMetricValue(spaceInfo["free_size"]),
	)

	if total <= 0 && used > 0 && remain > 0 {
		total = used + remain
	}
	if remain <= 0 && total > used && used > 0 {
		remain = total - used
	}

	if total > 0 {
		view.SpaceTotal = total
	}
	if used > 0 {
		view.SpaceUsed = used
	}
	if remain > 0 {
		view.SpaceRemain = remain
	}
}

func decodeObject(data []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func apiError(body map[string]any, fallback string) error {
	message := strings.TrimSpace(firstNonEmpty(
		asString(body["message"]),
		asString(body["error"]),
		asString(body["msg"]),
	))
	if message == "" {
		return errors.New(fallback)
	}
	return fmt.Errorf("%s: %s", fallback, message)
}

func isSuccessBody(body map[string]any) bool {
	switch value := body["state"].(type) {
	case bool:
		return value
	case float64:
		return value == 1
	case int:
		return value == 1
	case int64:
		return value == 1
	case string:
		return value == "1" || strings.EqualFold(value, "true")
	default:
		return false
	}
}

func credentialFromAny(value any) *config.Credential {
	switch typed := value.(type) {
	case map[string]any:
		return credentialFromMap(typed)
	case string:
		parsed := &driver.Credential{}
		if err := parsed.FromCookie(typed); err != nil {
			return nil
		}
		return &config.Credential{
			UID:  parsed.UID,
			CID:  parsed.CID,
			SEID: parsed.SEID,
			KID:  parsed.KID,
		}
	default:
		return nil
	}
}

func credentialFromMap(value map[string]any) *config.Credential {
	credential := &config.Credential{
		UID:  firstNonEmpty(asString(value["UID"]), asString(value["uid"])),
		CID:  firstNonEmpty(asString(value["CID"]), asString(value["cid"])),
		SEID: firstNonEmpty(asString(value["SEID"]), asString(value["seid"])),
		KID:  firstNonEmpty(asString(value["KID"]), asString(value["kid"])),
	}
	if !credentialValid(credential) {
		return nil
	}
	return credential
}

func credentialFromResponseCookies(resp *http.Response) *config.Credential {
	if resp == nil {
		return nil
	}

	items := map[string]any{}
	for _, cookie := range resp.Cookies() {
		if cookie == nil {
			continue
		}
		items[strings.ToUpper(cookie.Name)] = cookie.Value
	}
	return credentialFromMap(items)
}

func credentialValid(credential *config.Credential) bool {
	return credential != nil &&
		strings.TrimSpace(credential.UID) != "" &&
		strings.TrimSpace(credential.CID) != "" &&
		strings.TrimSpace(credential.SEID) != ""
}

func extractFaceURL(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		return firstNonEmpty(
			asString(typed["face_1"]),
			asString(typed["face_l"]),
			asString(typed["face_m"]),
			asString(typed["face_s"]),
			asString(typed["url"]),
		)
	default:
		return ""
	}
}

func asMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func asInt64(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	case json.Number:
		result, _ := typed.Int64()
		return result
	case string:
		result, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return result
	default:
		return 0
	}
}

func isTruthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		normalized := strings.TrimSpace(strings.ToLower(typed))
		return normalized == "1" || normalized == "true" || normalized == "yes"
	default:
		return asInt64(value) > 0
	}
}

func firstPositiveInt64(values ...any) int64 {
	for _, value := range values {
		if next := asInt64(value); next > 0 {
			return next
		}
	}
	return 0
}

func spaceMetricValue(value any) int64 {
	if metric := asInt64(value); metric > 0 {
		return metric
	}

	typed := asMap(value)
	return firstPositiveInt64(
		typed["size"],
		typed["value"],
		typed["num"],
		typed["bytes"],
		typed["count"],
	)
}

func formatUnixishTime(value int64) string {
	if value <= 0 {
		return ""
	}

	if value > 1_000_000_000_000 {
		value /= 1000
	}
	return time.Unix(value, 0).Format(timeLayoutRFC3339)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
