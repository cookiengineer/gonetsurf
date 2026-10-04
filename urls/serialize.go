package urls

import "strings"

// String returns the URL serialized per the URL Standard serializer.
func (u *URL) String() string {
	var output strings.Builder
	output.WriteString(u.Scheme)
	output.WriteByte(':')
	if u.HasHost {
		output.WriteString("//")
		if u.Username != "" || u.Password != "" {
			output.WriteString(u.Username)
			if u.Password != "" {
				output.WriteByte(':')
				output.WriteString(u.Password)
			}
			output.WriteByte('@')
		}
		output.WriteString(u.Host)
		if u.Port != "" {
			output.WriteByte(':')
			output.WriteString(u.Port)
		}
	} else if !u.CannotBeABase && len(u.Path) > 1 && u.Path[0] == "" {
		output.WriteString("/.")
	}
	if u.CannotBeABase {
		output.WriteString(u.OpaquePath)
	} else {
		for _, segment := range u.Path {
			output.WriteByte('/')
			output.WriteString(segment)
		}
	}
	if u.HasQuery {
		output.WriteByte('?')
		output.WriteString(u.Query)
	}
	if u.HasFragment {
		output.WriteByte('#')
		output.WriteString(u.Fragment)
	}
	return output.String()
}

// Protocol returns the scheme with a trailing colon.
func (u *URL) Protocol() string {
	return u.Scheme + ":"
}

// Hostname returns the serialized host without a port.
func (u *URL) Hostname() string {
	return u.Host
}

// HostWithPort returns the serialized host and a non-null port.
func (u *URL) HostWithPort() string {
	if u.Port == "" {
		return u.Host
	}
	return u.Host + ":" + u.Port
}

// Pathname returns the serialized path.
func (u *URL) Pathname() string {
	if u.CannotBeABase {
		return u.OpaquePath
	}
	var output strings.Builder
	for _, segment := range u.Path {
		output.WriteByte('/')
		output.WriteString(segment)
	}
	return output.String()
}

// Search returns the query including its leading question mark, or the empty
// string when the query is null or empty, matching the URL API.
func (u *URL) Search() string {
	if !u.HasQuery || u.Query == "" {
		return ""
	}
	return "?" + u.Query
}

// Hash returns the fragment including its leading hash sign, or the empty
// string when the fragment is null or empty, matching the URL API.
func (u *URL) Hash() string {
	if !u.HasFragment || u.Fragment == "" {
		return ""
	}
	return "#" + u.Fragment
}

// HasOpaquePath reports whether the URL has a path that cannot be a base.
func (u *URL) HasOpaquePath() bool {
	return u.CannotBeABase
}
