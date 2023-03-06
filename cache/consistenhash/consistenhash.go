package consistenhash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	//hash: 哈希函数
	hash Hash
	//keys: 哈希环
	keys []int
	//replicas: 虚拟节点倍数
	replicas int
	//虚拟节点与真实节点的映射表 key是虚拟节点哈希值 value是真实节点名称
	hashMap map[int]string
}

//New:初始化哈希环
func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

//Add:添加服务器真实节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			//虚拟节点为 strconv.Itoa(i) + key，即通过添加编号的方式区分不同虚拟节点
			dummyNode := []byte(strconv.Itoa(i) + key)
			hashValue := int(m.hash(dummyNode))
			m.keys = append(m.keys, hashValue)
			//增加虚拟节点与真实节点的映射关系
			m.hashMap[hashValue] = key
		}
	}
	sort.Ints(m.keys)
}

//获取key的对应真实节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	//Search函数采用二分法搜索找到[0, n)区间内最小的满足f(i)==true的值i
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
