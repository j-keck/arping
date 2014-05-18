# arping
  
arping is a native go library to ping a host per arp datagram, or query a host mac address 

The currently supported platforms are: Linux and BSD.
  

## Usage
### arping library

* import this library per `import "github.com/j-keck/arping"`
* export GOPATH if not already (`export GOPATH=$PWD`)
* download the library `go get`
* run it `sudo -E go run <YOUR PROGRAMM>` 
* or build it `go build`


The library requires raw socket access. So it must run as root, or with appropriate capabilities under linux: `sudo setcap cap_net_raw+ep <BIN>`.


#### Examples

##### ping a host:
     package main
     import ("fmt"; "github.com/j-keck/arping"; "net")

     func main(){
       dstIP := net.ParseIP("192.168.1.1")
       if hwAddr, duration, err := arping.Arping(dstIP); err != nil {
         fmt.Println(err)
       } else {
         fmt.Printf("%s (%s) %d usec\n", dstIP, hwAddr, duration/1000)
       }
     }

##### resolve mac address:
     package main
     import ("fmt"; "github.com/j-keck/arping"; "net")

     func main(){  
       dstIP := net.ParseIP("192.168.1.1")
       if hwAddr, _, err := arping.Arping(dstIP); err != nil {
         fmt.Println(err)
       } else {
         fmt.Printf("%s is at %s\n", dstIP, hwAddr)
       }
     }

##### check if host is online:
     package main
     import ("fmt"; "github.com/j-keck/arping"; "net")

     func main(){
       dstIP := net.ParseIP("192.168.1.1")
       _, _, err := arping.Arping(dstIP)
       if err == arping.ErrTimeout {
         fmt.Println("offline")
       }else if err != nil {
         fmt.Println(err.Error())
       }else{
         fmt.Println("online")
       }
     }
  

  
### arping executable
   
To get a runnable pinger use `go get -u github.com/j-keck/arping/cmd/arping`. This will build the binary in $GOPATH/bin.

arping requires raw socket access. So it must run as root, or with appropriate capabilities under Linux: `sudo setcap cap_net_raw+ep <ARPING_PATH>`.

