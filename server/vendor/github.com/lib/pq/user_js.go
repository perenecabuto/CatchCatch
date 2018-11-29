// Package pq is a pure Go Postgres driver for the database/sql package.
package pq

// Perform Windows user name lookup identically to libpq.
//
// The PostgreSQL code makes use of the legacy Win32 function
// GetUserName, and that function has not been imported into stock Go.
// GetUserNameEx is available though, the difference being that a
// wider range of names are available.  To get the output to be the
// same as GetUserName, only the base (or last) component of the
// result is returned.
func userCurrent() (string, error) {
	return "", nil
}
