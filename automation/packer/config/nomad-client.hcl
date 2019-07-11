data_dir  = "/var/nomad"

advertise {
  http = "192.168.50.5"
  rpc = "192.168.50.5"
  serf = "192.168.50.5"
}

client {
  enabled       = true
  servers = ["192.168.50.4"] 
  options {
    "driver.raw_exec.enable" = "1"
  }
  network_interface = "enp0s8"
}

consul {
  address = "192.168.50.4:8500"
}

