package operations

import "fmt"

func (a AlluxioFileUtils) AddWorker(podIP string) (err error) {

	var (
		str     = "echo echo " + podIP + ":1 >> /discover_hosts.sh"
		command = []string{"bash", "-c", str}
		stderr  string
		stdout  string
	)

	stdout, stderr, err = a.exec(command, false)
	if err != nil {
		err = fmt.Errorf("execute command %v with expectedErr: %v stdout %s and stderr %s", command, err, stdout, stderr)
		return
	}

	return
}

func (a AlluxioFileUtils) DeleteWorker(podIP string) (err error) {

	var (
		str     = " sed '/" + podIP + "/'d /discover_hosts.sh >  /discover_hosts.sh"
		command = []string{"bash", "-c", str}
		stderr  string
		stdout  string
	)

	stdout, stderr, err = a.exec(command, false)
	if err != nil {
		err = fmt.Errorf("execute command %v with expectedErr: %v stdout %s and stderr %s", command, err, stdout, stderr)
		return
	}

	return
}