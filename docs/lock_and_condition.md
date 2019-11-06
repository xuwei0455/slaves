## Lock and Condition


### mutex and spin lock
Golang的标准库里已经有一个不错的互斥锁实现了，即`sync.Mutex`，互斥锁会在其他goroutine去
`Lock`的时候让申请锁的goroutine进入`sleep`，互斥锁的使用很简单，这里就不赘述了。具体可
参考[官方手册](https://golang.org/pkg/sync/)


而在Golang中常用的还有`SpinLock`，但是golang的标准库里并没有实现自旋锁，但是在golang里
实现这样的一个锁还是很简单的，使用`CAS algorithm (compare and swap)`即可实现一个简单
的自旋锁。

```go
type spinLock uint32

func (sl *spinLock) Lock() {
    for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
        runtime.Gosched()
    }
}

func (sl *spinLock) Unlock() {   
    atomic.StoreUint32((*uint32)(sl), 0)
}

func NewSpinLock() sync.Locker {
    var lock spinLock 
    return &lock
}
```

不过这个锁不是可重入锁，为了支持再次加锁，需要一个计数器来记录已上锁的次数，如下：
```go
type spinLock struct {
    owner int
    count int
}

func (sl *spinLock) Lock() {
    if sl.owner == GetGID() {
        sl.count++
        return
    }
    
    for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
        runtime.Gosched()
    }
}

func (sl *spinLock) Unlock() {
    if sl.owner != GetGID() {
        panic("BUG: not own the spin lock")
    }

    if sl.count > 0 {
        sl.count--
    } else {
        atomic.StoreUint32((*uint32)(sl), 0)
    }
}

func GetGID() uint64 {
    b := make([]byte, 64)
    b = b[:runtime.Stack(b, false)]
    b = bytes.TrimPrefix(b, []byte("goroutine "))
    b = b[:bytes.IndexByte(b, ' ')]
    n, _ := strconv.ParseUint(string(b), 10, 64)
    return n
}
```

### condition

`Cond`也是`sync`包提供的一个goroutine之间同步的一个工具，可以通过它完成锁释放后的通知
逻辑，从`Cond`的接口来看可以很清晰的看到分成两个部分，`Wait`和`Signal`,`Boardcast`。
语义很清楚，分别是等待通知，单独通知，和广播通知。使用起来也是非常简单，不再赘述。需要
注意的是这个不是一个通知机制，而是等待锁同步的一个工具而已。

