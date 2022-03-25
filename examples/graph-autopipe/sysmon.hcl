# super-simple system monitor
connect = {
  monitor: ["stdout"]
}

sysmon "monitor" {
  interval= "5s"
}

stdout "stdout" {}

