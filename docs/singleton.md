## Singleton Pattern

单例模式在很多编程语言里都很常见，实现思路也无非是在创建实例的过程中判断一下是否已有
实例，如果有，直接返回；如果没有，则创建。

比较尴尬的是golang里别说是构造函数了，连真正意思上的类都没有，只有一个`struct`，一
般新建实例都是写个`New()`函数滥竽充数，不过既然有这么一个通用的规范，那当然还是要遵
循，所以很容易想到的就是在`New()`的时候判断是不是需要创建新的对象了。如下：

``` go
package singleton

type singleton struct {}

var instance *singleton

func GetInstance() *singleton {
    if instance == nil {
        instance = &singleton{}
    }
    return instance
}
```
golang是一个天生的高并发语言，很多时候编程都需要站在并发的角度去看问题，就如上这段
代码，很明显`if instance == nil`这里不是线程安全的（或者叫goroutine安全？）,
解决的办法也很简单，加锁就是了。所以有如下代码：
``` go
var mutex Sync.Mutex

func GetInstance() *singleton {
    mutex.Lock()
    defer mutex.Unlock()

    if instance == nil {
        instance = &singleton{}
    }
    return instance
}
```
这里其实带来了一定的性能问题，因为其实并不是每次都需要上锁的，因为当实例已经存在的情况
下，是不用上锁的。那么优化的措施也就很明显了，只有在实例不存在的时候，再去上锁确认实例
是否真正没有被创建。
``` go
var mutex Sync.Mutex

func GetInstance() *singleton {
    if instance == nil {
        mutex.Lock()
        defer mutex.Unlock()

        if instance == nil {
            instance = &singleton{}
        }
    }
    return instance
}
```
但这其实也并不是万无一失，因为instance的赋值和实例化并不能保证同时完成，编译器优化后，
可能在赋值结束后(即为变量分配了地址空间，但并没有将对象写入分配的空间)就离开了`if`块，
从而释放了锁。这个时候另一个goroutine进来就会读取到非空的地址，但是却指向了一块还没有
初始化的内存。（JAVA的机制是这样的，GO还不确定。可参考[Go内存模型](https://golang.org/ref/mem#tmp_9)）

当然Go需要提出一个解决办法，那就是sync模块，这是个同步模块。简单的来讲，所有要涉及到
同步的场景可以优先考虑使用这个包，那么改写后例子如下：

``` go
func GetInstance() *singleton {
    once.Do(func(){
        instance = &singleton{}
    })
    return instance
}
```

那once.Do做了什么呢？其实是调用了atomic包来保证func原子的被执行了，这样可以guarantee
其他所有goroutine的read一定在instance初始化完成之后。