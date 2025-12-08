package binaries

const (
	version    string = "1.0"
	currentSum string = "034387c16e87eef37f9c3a114a751661abab8c95c3811e6b36cc87c8c361abf0"
)

func FormatVersion() string {
	str := version + "\n" + currentSum
	return str
}
