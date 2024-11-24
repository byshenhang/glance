package define

// SharedSecret 是用于生成动态密钥的共享密钥。
// 请确保在生产环境中安全存储和管理此密钥。
const SharedSecret = `W.c!|-{Vo7+r#)Kb4cEQs17g7A@)^Dfk"?wb_MTat$c!v{Kl):klClm1EZyE&e`

// DefaultTimeStep 是动态密钥生成和验证的默认时间步长（秒）。
const DefaultTimeStep = 10

// Tolerance 是验证动态密钥时允许的时间窗口偏移量。
const Tolerance = 1

const CookieAPI = "http://127.0.0.1:8100/api/cookie"
