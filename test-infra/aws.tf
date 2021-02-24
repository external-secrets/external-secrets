// simple secret as StringValue
resource "aws_secretsmanager_secret" "simple_string" {
  name = "simple-string"
}

resource "aws_secretsmanager_secret_version" "simple_string" {
  secret_id     = aws_secretsmanager_secret.simple_string.id
  secret_string = file("${path.module}/data/simple")
}

// simple secret
resource "aws_secretsmanager_secret" "simple_binary" {
  name = "simple-binary"
}

resource "aws_secretsmanager_secret_version" "simple_binary" {
  secret_id     = aws_secretsmanager_secret.simple_binary.id
  secret_binary = filebase64("${path.module}/data/simple")
}

// json string
resource "aws_secretsmanager_secret" "json_string" {
  name = "json-string"
}

resource "aws_secretsmanager_secret_version" "json_string" {
  secret_id     = aws_secretsmanager_secret.json_string.id
  secret_string = file("${path.module}/data/secretjson")
}

// json binary
resource "aws_secretsmanager_secret" "json_bin" {
  name = "json-binary"
}

resource "aws_secretsmanager_secret_version" "json_bin" {
  secret_id     = aws_secretsmanager_secret.json_bin.id
  secret_binary = filebase64("${path.module}/data/secretjson")
}
