# Yun-Brute 
[![rcard](https://goreportcard.com/badge/github.com/hanbang-wang/Yun-Brute)](https://goreportcard.com/report/github.com/hanbang-wang/Yun-Brute)

A working brute-force machine for BaiduYun private share links.

# Example
![Example-GIF](https://www.superfashi.com/wp-content/uploads/2016/11/a.gif)

Try running this link on your own computer/server!

# Compilation

Firstly you have to `go get -u` two packages I used in this projext:

- `gopkg.in/alecthomas/kingpin.v2`
- `gopkg.in/cheggaaa/pb.v1`

Then clone this project to run.
```bash
git clone https://github.com/hanbang-wang/Yun-Brute
go run brute.go
```
Or simply uses pre-compiled binaries downloaded [here](https://github.com/hanbang-wang/Yun-Brute/releases).

# Usage
```bash
brute [<flags>] <link>

Flags:
  -h, --help            Show context-sensitive help.
  -p, --preset="0000"   The preset start of key to brute.
  -t, --thread=1000     Number of threads.

Args:
  <link> URL of BaiduYun file you want to get.
```

# Feature
- **Resolver**  
You can use 2 types of BaiduYun links in the out-of-the-box version. If there's more types of private share links, you can add the resolver yourself, or let me know by sending a PR or taking an issue.  

- **Interrupt-friendly:**  
When you press `Ctrl-C` to interrupt the program, it will output the current progress, which allow you to use `-p` flag to continue working later.

- **Logging:**  
Unfortunately the logging would mess up with the progress bar, use `2> /dev/null` to disable logging, or you can try `1>&2`.
 
- **Proxying:**  
This program contains 4 methods of acquiring proxies, with repetition and failure proxies correction. When there're no proxies left, threads will auto hang up to wait for more proxies to come in. And you can add your own source of proxies easily.

# License
This little project uses `MIT License`, for more info please visit [LICENSE](LICENSE).
