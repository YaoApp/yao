package crypto

import (
	"crypto"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestMD4(t *testing.T) {
	// Hash
	args := []interface{}{"MD4", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "e10adc3949ba59abbe56e057f20f883e", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "30ce71a73bdd908c3955a90e8f7429ef", res)
}

func TestMD5(t *testing.T) {
	// Hash
	args := []interface{}{"MD5", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "e10adc3949ba59abbe56e057f20f883e", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "30ce71a73bdd908c3955a90e8f7429ef", res)
}

func TestSHA1(t *testing.T) {
	// Hash
	args := []interface{}{"SHA1", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "7c4a8d09ca3762af61e59520943dc26494f8941b", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "74b55b6ab2b8e438ac810435e369e3047b3951d0", res)
}

func TestSHA256(t *testing.T) {
	// Hash
	args := []interface{}{"SHA256", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "b8ad08a3a547e35829b821b75370301dd8c4b06bdd7771f9b541a75914068718", res)
}

func TestSHA1Base64(t *testing.T) {
	// Hash
	args := []interface{}{"SHA1", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "7c4a8d09ca3762af61e59520943dc26494f8941b", res)

	// HMac
	args = append(args, "123456", "base64")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "dLVbarK45DisgQQ142njBHs5UdA=", res)
}

func TestRSA2Sign(t *testing.T) {
	prikey := `MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCHwr1gmVkw1pp4+DP74J+l4c9GUyySjIsBECspMDX83Au/OmZ5o1IxCg95rGzAC5W908J084seOvVJcLmFY5H2w6pHSyqho/OLTxupH0jN+wRQeLRIwDvyFZWIYODk8eAktpBpphgq3hL/NG7P87tuAoWIiJ1w8lNW85FqTLKpgvtfFmCL3jdSZwgLEbS3up7WM12hNU4pakKWdlPwse9rCFFTiR/Qm1eNzyzz4cGX5M1FMW8ByxXd5l6PSGR53wJPGiwv5kvsudjKXvRw4tqUgNIsmtzg/xBDMrbX6E6HqsB6UfTUQNM4FT3g7UhcT0D+BpvHNcCSupZcvYm9aN3LAgMBAAECggEADcLUlV0V6FhocgiepFJhfFwGOZemtfgfAu2TomornsTjP+/4gS3n3+aoKOosX88Mz6AOXvJs0JSjVl1hwL6WBhBRS0a4PIg04JMVN7BfHdnq1wlVJOavbNt5O8iuIybNVItY2gym+HloLYwwC04mWoFQ7cUDSHaXsgGgZMj/dyUUbio0KdLsWGot9ajDX4Det6D97pl+KpaT3Yz1JrOaen/iCpZ5tMRN7kDAyVzGJqn9++Hu0+lgVm7eVEF8ny6BALObKgEvhMT7U0O9/lVXgz2ZnyqOqAhzXsm9MeQfpgTAphnUOwPJDaDo9K7tM9PHYiwkbV7C05OEmSS9YTeOAQKBgQDbpuEjgGzcXp+6SSAkRmaVeAh+VUB/JIWbdY/6U+f7E/qM4UgnBJubjyMYCN7+uGICzCbBdXQk8zNZOTeuhD0yI46RXQyqlkhkzLWNuIBAph8L2dmxNhH1biVjvauPo2WLhIygn33Yd3eh/h73jmzFvbB3DL82Dp9JXrOIMRGKywKBgQCeOfm5mDbjb8UN3qoJ5oJjSyQ46RfPIbCmMt1h6TeB9XbztnuJVs7hn7DvkkcHVgtq3ipdyHL8fDTSbJ3Mek84wEYgyuXnPsMlwGyUiaCJLwrXSdh9/4KmjrfZw6vdciW8MPvExzNtYinSZIZ8yMKQmkLaGfMzN5kKJN8EcKyZAQKBgA16BrQ76/H1aE1wsSUooKCpFbRSnLtwTTZFl0jfnwsbpbLBG8ExGi8IMDoISU5Nl83eIr6Z6z9dIJhn10/A01RhNB0dHWrV/6kXmkgQuuW8i4kZm66wx5dMY8Tj3UPZ3aAayNoODxWZ9uAcjF/aADh9s/cJ9C1n5kQFKHTBtfbTAoGAY/HxGVfZy/5M9b7hn5FYaUoMnlo2bOM2BzV3+6HqKxAXTEjHbfBEi+ZoSFwYu7yRR7cAAe9dGrmGUCjF4GSd6BYj9hDT+ib987nBnG321tC9Q1JlCum76GOcJFTiGeZBicdTMXA2vvBTxI81GFtj8x1N/yCHK6IB7JNvwAlALQECgYAo5iMhlQk+IjuilQnzKH9r3pCyhu/MYKtlvQYu5cg1lVbyU8fpn0FHdnglxErWIXWz5w5E9Q0mtdtL9T/89DDXNM7eue6PvgHJVmUTTIUkl85gGKyefSHTT57L9h3elMGPVNAG14qfyCeDQ6vJg1+VLSUWXwQ5e3DTuZL9wDe/ZA==`
	hash := HashTypes["SHA256"]
	res, err := RSA2Sign(prikey, hash, "hello world")
	if err != nil {
		t.Fatal(err)
	}

	pubKey := `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAh8K9YJlZMNaaePgz++CfpeHPRlMskoyLARArKTA1/NwLvzpmeaNSMQoPeaxswAuVvdPCdPOLHjr1SXC5hWOR9sOqR0sqoaPzi08bqR9IzfsEUHi0SMA78hWViGDg5PHgJLaQaaYYKt4S/zRuz/O7bgKFiIidcPJTVvORakyyqYL7XxZgi943UmcICxG0t7qe1jNdoTVOKWpClnZT8LHvawhRU4kf0JtXjc8s8+HBl+TNRTFvAcsV3eZej0hked8CTxosL+ZL7LnYyl70cOLalIDSLJrc4P8QQzK21+hOh6rAelH01EDTOBU94O1IXE9A/gabxzXAkrqWXL2JvWjdywIDAQAB`
	valid, err := RSA2Verify(pubKey, hash, "hello world", res)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, valid)
}

func TestRSA2SignSHA1(t *testing.T) {
	prikey := `MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCHwr1gmVkw1pp4+DP74J+l4c9GUyySjIsBECspMDX83Au/OmZ5o1IxCg95rGzAC5W908J084seOvVJcLmFY5H2w6pHSyqho/OLTxupH0jN+wRQeLRIwDvyFZWIYODk8eAktpBpphgq3hL/NG7P87tuAoWIiJ1w8lNW85FqTLKpgvtfFmCL3jdSZwgLEbS3up7WM12hNU4pakKWdlPwse9rCFFTiR/Qm1eNzyzz4cGX5M1FMW8ByxXd5l6PSGR53wJPGiwv5kvsudjKXvRw4tqUgNIsmtzg/xBDMrbX6E6HqsB6UfTUQNM4FT3g7UhcT0D+BpvHNcCSupZcvYm9aN3LAgMBAAECggEADcLUlV0V6FhocgiepFJhfFwGOZemtfgfAu2TomornsTjP+/4gS3n3+aoKOosX88Mz6AOXvJs0JSjVl1hwL6WBhBRS0a4PIg04JMVN7BfHdnq1wlVJOavbNt5O8iuIybNVItY2gym+HloLYwwC04mWoFQ7cUDSHaXsgGgZMj/dyUUbio0KdLsWGot9ajDX4Det6D97pl+KpaT3Yz1JrOaen/iCpZ5tMRN7kDAyVzGJqn9++Hu0+lgVm7eVEF8ny6BALObKgEvhMT7U0O9/lVXgz2ZnyqOqAhzXsm9MeQfpgTAphnUOwPJDaDo9K7tM9PHYiwkbV7C05OEmSS9YTeOAQKBgQDbpuEjgGzcXp+6SSAkRmaVeAh+VUB/JIWbdY/6U+f7E/qM4UgnBJubjyMYCN7+uGICzCbBdXQk8zNZOTeuhD0yI46RXQyqlkhkzLWNuIBAph8L2dmxNhH1biVjvauPo2WLhIygn33Yd3eh/h73jmzFvbB3DL82Dp9JXrOIMRGKywKBgQCeOfm5mDbjb8UN3qoJ5oJjSyQ46RfPIbCmMt1h6TeB9XbztnuJVs7hn7DvkkcHVgtq3ipdyHL8fDTSbJ3Mek84wEYgyuXnPsMlwGyUiaCJLwrXSdh9/4KmjrfZw6vdciW8MPvExzNtYinSZIZ8yMKQmkLaGfMzN5kKJN8EcKyZAQKBgA16BrQ76/H1aE1wsSUooKCpFbRSnLtwTTZFl0jfnwsbpbLBG8ExGi8IMDoISU5Nl83eIr6Z6z9dIJhn10/A01RhNB0dHWrV/6kXmkgQuuW8i4kZm66wx5dMY8Tj3UPZ3aAayNoODxWZ9uAcjF/aADh9s/cJ9C1n5kQFKHTBtfbTAoGAY/HxGVfZy/5M9b7hn5FYaUoMnlo2bOM2BzV3+6HqKxAXTEjHbfBEi+ZoSFwYu7yRR7cAAe9dGrmGUCjF4GSd6BYj9hDT+ib987nBnG321tC9Q1JlCum76GOcJFTiGeZBicdTMXA2vvBTxI81GFtj8x1N/yCHK6IB7JNvwAlALQECgYAo5iMhlQk+IjuilQnzKH9r3pCyhu/MYKtlvQYu5cg1lVbyU8fpn0FHdnglxErWIXWz5w5E9Q0mtdtL9T/89DDXNM7eue6PvgHJVmUTTIUkl85gGKyefSHTT57L9h3elMGPVNAG14qfyCeDQ6vJg1+VLSUWXwQ5e3DTuZL9wDe/ZA==`
	hash := HashTypes["SHA1"]
	res, err := RSA2Sign(prikey, hash, "hello world")
	if err != nil {
		t.Fatal(err)
	}

	pubKey := `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAh8K9YJlZMNaaePgz++CfpeHPRlMskoyLARArKTA1/NwLvzpmeaNSMQoPeaxswAuVvdPCdPOLHjr1SXC5hWOR9sOqR0sqoaPzi08bqR9IzfsEUHi0SMA78hWViGDg5PHgJLaQaaYYKt4S/zRuz/O7bgKFiIidcPJTVvORakyyqYL7XxZgi943UmcICxG0t7qe1jNdoTVOKWpClnZT8LHvawhRU4kf0JtXjc8s8+HBl+TNRTFvAcsV3eZej0hked8CTxosL+ZL7LnYyl70cOLalIDSLJrc4P8QQzK21+hOh6rAelH01EDTOBU94O1IXE9A/gabxzXAkrqWXL2JvWjdywIDAQAB`
	valid, err := RSA2Verify(pubKey, hash, "hello world", res)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, valid)
}

func TestRSA2SignBase64(t *testing.T) {
	prikey := `MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCHwr1gmVkw1pp4+DP74J+l4c9GUyySjIsBECspMDX83Au/OmZ5o1IxCg95rGzAC5W908J084seOvVJcLmFY5H2w6pHSyqho/OLTxupH0jN+wRQeLRIwDvyFZWIYODk8eAktpBpphgq3hL/NG7P87tuAoWIiJ1w8lNW85FqTLKpgvtfFmCL3jdSZwgLEbS3up7WM12hNU4pakKWdlPwse9rCFFTiR/Qm1eNzyzz4cGX5M1FMW8ByxXd5l6PSGR53wJPGiwv5kvsudjKXvRw4tqUgNIsmtzg/xBDMrbX6E6HqsB6UfTUQNM4FT3g7UhcT0D+BpvHNcCSupZcvYm9aN3LAgMBAAECggEADcLUlV0V6FhocgiepFJhfFwGOZemtfgfAu2TomornsTjP+/4gS3n3+aoKOosX88Mz6AOXvJs0JSjVl1hwL6WBhBRS0a4PIg04JMVN7BfHdnq1wlVJOavbNt5O8iuIybNVItY2gym+HloLYwwC04mWoFQ7cUDSHaXsgGgZMj/dyUUbio0KdLsWGot9ajDX4Det6D97pl+KpaT3Yz1JrOaen/iCpZ5tMRN7kDAyVzGJqn9++Hu0+lgVm7eVEF8ny6BALObKgEvhMT7U0O9/lVXgz2ZnyqOqAhzXsm9MeQfpgTAphnUOwPJDaDo9K7tM9PHYiwkbV7C05OEmSS9YTeOAQKBgQDbpuEjgGzcXp+6SSAkRmaVeAh+VUB/JIWbdY/6U+f7E/qM4UgnBJubjyMYCN7+uGICzCbBdXQk8zNZOTeuhD0yI46RXQyqlkhkzLWNuIBAph8L2dmxNhH1biVjvauPo2WLhIygn33Yd3eh/h73jmzFvbB3DL82Dp9JXrOIMRGKywKBgQCeOfm5mDbjb8UN3qoJ5oJjSyQ46RfPIbCmMt1h6TeB9XbztnuJVs7hn7DvkkcHVgtq3ipdyHL8fDTSbJ3Mek84wEYgyuXnPsMlwGyUiaCJLwrXSdh9/4KmjrfZw6vdciW8MPvExzNtYinSZIZ8yMKQmkLaGfMzN5kKJN8EcKyZAQKBgA16BrQ76/H1aE1wsSUooKCpFbRSnLtwTTZFl0jfnwsbpbLBG8ExGi8IMDoISU5Nl83eIr6Z6z9dIJhn10/A01RhNB0dHWrV/6kXmkgQuuW8i4kZm66wx5dMY8Tj3UPZ3aAayNoODxWZ9uAcjF/aADh9s/cJ9C1n5kQFKHTBtfbTAoGAY/HxGVfZy/5M9b7hn5FYaUoMnlo2bOM2BzV3+6HqKxAXTEjHbfBEi+ZoSFwYu7yRR7cAAe9dGrmGUCjF4GSd6BYj9hDT+ib987nBnG321tC9Q1JlCum76GOcJFTiGeZBicdTMXA2vvBTxI81GFtj8x1N/yCHK6IB7JNvwAlALQECgYAo5iMhlQk+IjuilQnzKH9r3pCyhu/MYKtlvQYu5cg1lVbyU8fpn0FHdnglxErWIXWz5w5E9Q0mtdtL9T/89DDXNM7eue6PvgHJVmUTTIUkl85gGKyefSHTT57L9h3elMGPVNAG14qfyCeDQ6vJg1+VLSUWXwQ5e3DTuZL9wDe/ZA==`
	hash := HashTypes["SHA256"]
	res, err := RSA2Sign(prikey, hash, "hello world", "base64")
	if err != nil {
		t.Fatal(err)
	}

	pubKey := `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAh8K9YJlZMNaaePgz++CfpeHPRlMskoyLARArKTA1/NwLvzpmeaNSMQoPeaxswAuVvdPCdPOLHjr1SXC5hWOR9sOqR0sqoaPzi08bqR9IzfsEUHi0SMA78hWViGDg5PHgJLaQaaYYKt4S/zRuz/O7bgKFiIidcPJTVvORakyyqYL7XxZgi943UmcICxG0t7qe1jNdoTVOKWpClnZT8LHvawhRU4kf0JtXjc8s8+HBl+TNRTFvAcsV3eZej0hked8CTxosL+ZL7LnYyl70cOLalIDSLJrc4P8QQzK21+hOh6rAelH01EDTOBU94O1IXE9A/gabxzXAkrqWXL2JvWjdywIDAQAB`
	valid, err := RSA2Verify(pubKey, hash, "hello world", res, "base64")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, valid)
}

func TestRSA2SignProcess(t *testing.T) {

	prikey := `MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCHwr1gmVkw1pp4+DP74J+l4c9GUyySjIsBECspMDX83Au/OmZ5o1IxCg95rGzAC5W908J084seOvVJcLmFY5H2w6pHSyqho/OLTxupH0jN+wRQeLRIwDvyFZWIYODk8eAktpBpphgq3hL/NG7P87tuAoWIiJ1w8lNW85FqTLKpgvtfFmCL3jdSZwgLEbS3up7WM12hNU4pakKWdlPwse9rCFFTiR/Qm1eNzyzz4cGX5M1FMW8ByxXd5l6PSGR53wJPGiwv5kvsudjKXvRw4tqUgNIsmtzg/xBDMrbX6E6HqsB6UfTUQNM4FT3g7UhcT0D+BpvHNcCSupZcvYm9aN3LAgMBAAECggEADcLUlV0V6FhocgiepFJhfFwGOZemtfgfAu2TomornsTjP+/4gS3n3+aoKOosX88Mz6AOXvJs0JSjVl1hwL6WBhBRS0a4PIg04JMVN7BfHdnq1wlVJOavbNt5O8iuIybNVItY2gym+HloLYwwC04mWoFQ7cUDSHaXsgGgZMj/dyUUbio0KdLsWGot9ajDX4Det6D97pl+KpaT3Yz1JrOaen/iCpZ5tMRN7kDAyVzGJqn9++Hu0+lgVm7eVEF8ny6BALObKgEvhMT7U0O9/lVXgz2ZnyqOqAhzXsm9MeQfpgTAphnUOwPJDaDo9K7tM9PHYiwkbV7C05OEmSS9YTeOAQKBgQDbpuEjgGzcXp+6SSAkRmaVeAh+VUB/JIWbdY/6U+f7E/qM4UgnBJubjyMYCN7+uGICzCbBdXQk8zNZOTeuhD0yI46RXQyqlkhkzLWNuIBAph8L2dmxNhH1biVjvauPo2WLhIygn33Yd3eh/h73jmzFvbB3DL82Dp9JXrOIMRGKywKBgQCeOfm5mDbjb8UN3qoJ5oJjSyQ46RfPIbCmMt1h6TeB9XbztnuJVs7hn7DvkkcHVgtq3ipdyHL8fDTSbJ3Mek84wEYgyuXnPsMlwGyUiaCJLwrXSdh9/4KmjrfZw6vdciW8MPvExzNtYinSZIZ8yMKQmkLaGfMzN5kKJN8EcKyZAQKBgA16BrQ76/H1aE1wsSUooKCpFbRSnLtwTTZFl0jfnwsbpbLBG8ExGi8IMDoISU5Nl83eIr6Z6z9dIJhn10/A01RhNB0dHWrV/6kXmkgQuuW8i4kZm66wx5dMY8Tj3UPZ3aAayNoODxWZ9uAcjF/aADh9s/cJ9C1n5kQFKHTBtfbTAoGAY/HxGVfZy/5M9b7hn5FYaUoMnlo2bOM2BzV3+6HqKxAXTEjHbfBEi+ZoSFwYu7yRR7cAAe9dGrmGUCjF4GSd6BYj9hDT+ib987nBnG321tC9Q1JlCum76GOcJFTiGeZBicdTMXA2vvBTxI81GFtj8x1N/yCHK6IB7JNvwAlALQECgYAo5iMhlQk+IjuilQnzKH9r3pCyhu/MYKtlvQYu5cg1lVbyU8fpn0FHdnglxErWIXWz5w5E9Q0mtdtL9T/89DDXNM7eue6PvgHJVmUTTIUkl85gGKyefSHTT57L9h3elMGPVNAG14qfyCeDQ6vJg1+VLSUWXwQ5e3DTuZL9wDe/ZA==`
	args := []interface{}{prikey, "SHA256", "hello world"}
	sign, err := process.New("crypto.RSA2Sign", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	pubKey := `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAh8K9YJlZMNaaePgz++CfpeHPRlMskoyLARArKTA1/NwLvzpmeaNSMQoPeaxswAuVvdPCdPOLHjr1SXC5hWOR9sOqR0sqoaPzi08bqR9IzfsEUHi0SMA78hWViGDg5PHgJLaQaaYYKt4S/zRuz/O7bgKFiIidcPJTVvORakyyqYL7XxZgi943UmcICxG0t7qe1jNdoTVOKWpClnZT8LHvawhRU4kf0JtXjc8s8+HBl+TNRTFvAcsV3eZej0hked8CTxosL+ZL7LnYyl70cOLalIDSLJrc4P8QQzK21+hOh6rAelH01EDTOBU94O1IXE9A/gabxzXAkrqWXL2JvWjdywIDAQAB`
	args = []interface{}{pubKey, "SHA256", "hello world", sign}
	valid, err := process.New("crypto.RSA2Verify", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, valid)
}

func TestRSA2SignProcessBase64(t *testing.T) {

	prikey := `MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCHwr1gmVkw1pp4+DP74J+l4c9GUyySjIsBECspMDX83Au/OmZ5o1IxCg95rGzAC5W908J084seOvVJcLmFY5H2w6pHSyqho/OLTxupH0jN+wRQeLRIwDvyFZWIYODk8eAktpBpphgq3hL/NG7P87tuAoWIiJ1w8lNW85FqTLKpgvtfFmCL3jdSZwgLEbS3up7WM12hNU4pakKWdlPwse9rCFFTiR/Qm1eNzyzz4cGX5M1FMW8ByxXd5l6PSGR53wJPGiwv5kvsudjKXvRw4tqUgNIsmtzg/xBDMrbX6E6HqsB6UfTUQNM4FT3g7UhcT0D+BpvHNcCSupZcvYm9aN3LAgMBAAECggEADcLUlV0V6FhocgiepFJhfFwGOZemtfgfAu2TomornsTjP+/4gS3n3+aoKOosX88Mz6AOXvJs0JSjVl1hwL6WBhBRS0a4PIg04JMVN7BfHdnq1wlVJOavbNt5O8iuIybNVItY2gym+HloLYwwC04mWoFQ7cUDSHaXsgGgZMj/dyUUbio0KdLsWGot9ajDX4Det6D97pl+KpaT3Yz1JrOaen/iCpZ5tMRN7kDAyVzGJqn9++Hu0+lgVm7eVEF8ny6BALObKgEvhMT7U0O9/lVXgz2ZnyqOqAhzXsm9MeQfpgTAphnUOwPJDaDo9K7tM9PHYiwkbV7C05OEmSS9YTeOAQKBgQDbpuEjgGzcXp+6SSAkRmaVeAh+VUB/JIWbdY/6U+f7E/qM4UgnBJubjyMYCN7+uGICzCbBdXQk8zNZOTeuhD0yI46RXQyqlkhkzLWNuIBAph8L2dmxNhH1biVjvauPo2WLhIygn33Yd3eh/h73jmzFvbB3DL82Dp9JXrOIMRGKywKBgQCeOfm5mDbjb8UN3qoJ5oJjSyQ46RfPIbCmMt1h6TeB9XbztnuJVs7hn7DvkkcHVgtq3ipdyHL8fDTSbJ3Mek84wEYgyuXnPsMlwGyUiaCJLwrXSdh9/4KmjrfZw6vdciW8MPvExzNtYinSZIZ8yMKQmkLaGfMzN5kKJN8EcKyZAQKBgA16BrQ76/H1aE1wsSUooKCpFbRSnLtwTTZFl0jfnwsbpbLBG8ExGi8IMDoISU5Nl83eIr6Z6z9dIJhn10/A01RhNB0dHWrV/6kXmkgQuuW8i4kZm66wx5dMY8Tj3UPZ3aAayNoODxWZ9uAcjF/aADh9s/cJ9C1n5kQFKHTBtfbTAoGAY/HxGVfZy/5M9b7hn5FYaUoMnlo2bOM2BzV3+6HqKxAXTEjHbfBEi+ZoSFwYu7yRR7cAAe9dGrmGUCjF4GSd6BYj9hDT+ib987nBnG321tC9Q1JlCum76GOcJFTiGeZBicdTMXA2vvBTxI81GFtj8x1N/yCHK6IB7JNvwAlALQECgYAo5iMhlQk+IjuilQnzKH9r3pCyhu/MYKtlvQYu5cg1lVbyU8fpn0FHdnglxErWIXWz5w5E9Q0mtdtL9T/89DDXNM7eue6PvgHJVmUTTIUkl85gGKyefSHTT57L9h3elMGPVNAG14qfyCeDQ6vJg1+VLSUWXwQ5e3DTuZL9wDe/ZA==`
	args := []interface{}{prikey, "SHA256", "hello world", "base64"}
	sign, err := process.New("crypto.RSA2Sign", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	pubKey := `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAh8K9YJlZMNaaePgz++CfpeHPRlMskoyLARArKTA1/NwLvzpmeaNSMQoPeaxswAuVvdPCdPOLHjr1SXC5hWOR9sOqR0sqoaPzi08bqR9IzfsEUHi0SMA78hWViGDg5PHgJLaQaaYYKt4S/zRuz/O7bgKFiIidcPJTVvORakyyqYL7XxZgi943UmcICxG0t7qe1jNdoTVOKWpClnZT8LHvawhRU4kf0JtXjc8s8+HBl+TNRTFvAcsV3eZej0hked8CTxosL+ZL7LnYyl70cOLalIDSLJrc4P8QQzK21+hOh6rAelH01EDTOBU94O1IXE9A/gabxzXAkrqWXL2JvWjdywIDAQAB`
	args = []interface{}{pubKey, "SHA256", "hello world", sign, "base64"}
	valid, err := process.New("crypto.RSA2Verify", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, valid)
}

// ProcessHmacWith tests

func TestHmacWith(t *testing.T) {
	keyhex := hex.EncodeToString([]byte("key"))
	valuehex := hex.EncodeToString([]byte("value"))
	keybase64 := base64.StdEncoding.EncodeToString([]byte("key"))
	valuebase64 := base64.StdEncoding.EncodeToString([]byte("value"))

	tests := []struct {
		name    string
		option  *hmacOption
		hash    crypto.Hash
		algo    string
		value   string
		key     string
		wantErr bool
	}{
		{
			name: "Test with hex encoding",
			option: &hmacOption{
				keyEncoding:    "hex",
				valueEncoding:  "hex",
				outputEncoding: "hex",
			},
			hash:    crypto.SHA256,
			algo:    "SHA256",
			value:   valuehex,
			key:     keyhex,
			wantErr: false,
		},
		{
			name: "Test with base64 encoding",
			option: &hmacOption{
				keyEncoding:    "base64",
				valueEncoding:  "base64",
				outputEncoding: "base64",
			},
			hash:    crypto.SHA256,
			value:   valuebase64,
			key:     keybase64,
			algo:    "SHA1",
			wantErr: false,
		},
		{
			name:    "Test with default encoding",
			option:  &hmacOption{},
			hash:    crypto.SHA256,
			value:   "value",
			key:     "key",
			wantErr: false,
		},
		{
			name:    "Test with nil option",
			option:  nil,
			hash:    crypto.SHA256,
			value:   "value",
			key:     "key",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HmacWith(tt.option, tt.hash, tt.value, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("HmacWith() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			option := map[string]interface{}{}
			if tt.option != nil {
				option = map[string]interface{}{
					"key":    tt.option.keyEncoding,
					"value":  tt.option.valueEncoding,
					"output": tt.option.outputEncoding,
					"algo":   tt.algo,
				}
			}
			args := []interface{}{option, tt.value, tt.key}
			_, err := process.New("crypto.HmacWith", args...).Exec()
			if (err != nil) != tt.wantErr {
				t.Errorf("HmacWith() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
