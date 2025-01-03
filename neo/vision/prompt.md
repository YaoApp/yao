根据这个数据结构，和说明实现一下对应的逻辑。

1. driver 单独一个目录. 每个 driver 一个目录，model 和 storage 在一级即可。 放在@vision 下
2. model driver: 支持 openai
3. storage driver 支持 local 和 s3 . local 使用我框架的 fs 实现，参考@file.go
4. 在 程序启动时候，设置视觉配置。(作为可选配置） @types.go
5. 统一的创建和调用入口，调用时需传入 chat model (用来判断是否支持视觉) 和图片路径，（使用 fs 读取）。 @vision.go
6. 用一个回调函数，外部传入用来格式化返回的数据。
