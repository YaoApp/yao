package otp

// Revoke removes an OTP code from the store immediately.
// It is silent when the code does not exist or has already expired.
func (s *Service) Revoke(code string) error {
	if code == "" {
		return nil
	}
	key := s.storeKey(code)
	_ = s.store.Del(key)
	return nil
}
