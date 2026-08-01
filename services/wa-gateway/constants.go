package main

// Redis key prefixes
const (
	redisKeyPWResetOTP  = "pw-reset-otp:"
	redisKeyAuthPending = "auth:pending:"
)

// Common messages
const (
	msgBatalCommand         = "Ketik *batal* untuk membatalkan."
	msgResetPasswordCanceled = "✅ Reset password dibatalkan."
)
