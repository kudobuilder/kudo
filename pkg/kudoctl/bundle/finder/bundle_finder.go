package finder

// Finder is a bundle finder and is any implementation which can find/discover a bundle
type Finder interface {
	GetBundle(name string) error
}

// LocalFinder will find local operator bundle: folders or tgz
type LocalFinder struct {
}

// URLFinder will find an operator bundle from a url
type URLFinder struct {
}
