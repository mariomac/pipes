# HCL Map defining connection between stage by their name
connect = {
  server: ["rawPrinter", "secureFilter"]
  secureFilter: ["safePrinter"]
}

# Here comes the definition of each graph stage

# Each graph stage is the type of the stage (http, stdout, deleter)
# followed by the name of the stage (there might be multiple stage of the same type)
# Inside, any configuration arguments for the stage should go

# http is listening for POST JSONs and forwards them to the next stage of the
# pipeline
http "server" {
  port = 8080
}

# stdout receives a string and prints it to the standard output, prepending the
# configured message
stdout "rawPrinter" {
  prepend = "Received message: "
}

stdout "safePrinter" {
  prepend = "Safe-to-show message: "
}

# deleter receives a map and removes all the indicated fields from it, then forwards it
deleter "secureFilter" {
  fields = ["password", "secret"]
}
