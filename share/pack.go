package share

import "io"

// ********************************************************************************
// WARNING: DO NOT MODIFY THIS FILE. IT WILL BE REPLACED BY THE APPLICATION CODE.
// *********************************************************************************

// Pack the yao app package
type Pack struct{}

// Encrypt encrypt
func (pkg *Pack) Encrypt(reader io.Reader, writer io.Writer) error {
	_, err := io.Copy(writer, reader)
	return err
}

// Decrypt decrypt
func (pkg *Pack) Decrypt(reader io.Reader, writer io.Writer) error {
	_, err := io.Copy(writer, reader)
	return err
}
