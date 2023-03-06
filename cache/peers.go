package cache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type PeerPicker interface {
	//PickPeer():用于根据传入的 key 选择相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

//HTTP客户端
type PeerGetter interface {
	//Get():方法用于从对应 group 查找缓存值
	Get(group string, key string) ([]byte, error)
}

//httpGetter: HTTP客户端
type httpGetter struct {
	//baseURL为将要访问的远程节点地址
	//例如 http://example.com/_whitecache/
	baseURL string
}

//从对应的group中获取值
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		//QueryEscape函数对s进行转码使之可以安全的用在URL查询里。
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	//使用http.get方法获取返回值
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned:%v", res.Status)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body:%v", bytes)
	}
	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)
