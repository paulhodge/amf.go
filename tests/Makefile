
protocol:
	cd .. && 6g protocol.go

tcp_listener: protocol
	6g tcp_listener.go
	6l -o tcp_listener tcp_listener.6
	./tcp_listener

remoting:
	mxmlc -services "services-config.xml" Remoting.mxml

socket:
	mxmlc Socket.mxml

sp_server:
	6g socket_policy_server.go && 6l -o socket_policy_server socket_policy_server.6
	echo "Launching with sudo b/c we need to listen to a port below 1024.."
	sudo ./socket_policy_server
