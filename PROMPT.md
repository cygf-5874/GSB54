第一步先跑 cd ignorepath && go run ./ignorecheck -list，看看现在的状态。
这是 Go 1.24 的 ignore 规则匹配库，只用标准库，go.mod 里没有任何 require。
README「对外契约」一节列了 9 条；ignorepath.go 里 6 个函数是空壳，
ignorepath_test.go 里 10 个测试函数当前全红（能编译，跑起来报未实现）。

任务：实现这 6 个函数，让 9 条契约同时成立。

验收（ignorecheck/ 是固定验收程序，别改）：
- go run ./ignorecheck 退出码 0，10 个场景全过（syntax 3 + negate 3 + ancestor 2 + robust 2）；
- go test ./... 全绿。

约束：
1. 不改 ignorecheck/，不改 ignorepath_test.go 里既有用例的断言。
2. 常量名与值不变；6 个函数的签名不变；errors.go 的哨兵错误不变。
3. 只看路径字符串，不读文件系统、不读 .gitignore 文件本身。
4. 不引入第三方依赖，只用标准库。
