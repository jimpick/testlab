data_dir  = "/var/nomad"

advertise {
  http = "192.168.50.4"
  rpc = "192.168.50.4"
  serf = "192.168.50.4"
}

server {
  enabled          = true
  bootstrap_expect = 1
}

client {
  enabled       = true
  options {
    "driver.raw_exec.enable" = "1"
  }
  network_interface = "enp0s8"
  meta {
    "first" = "1"
  }
}

consul {
  address = "127.0.0.1:8500"
}

