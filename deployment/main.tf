terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.40.0"
    }
  }
  required_version = ">= 1.2.0"
}

provider "aws" {
  region = var.aws_region
}

# DynamoDB table for parking tickets
resource "aws_dynamodb_table" "parking_tickets" {
  name         = "parkingTickets"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "ticketId"

  attribute {
    name = "ticketId"
    type = "S"
  }
  
  attribute {
    name = "plate"
    type = "S"
  }
  
  attribute {
    name = "parkingLot"
    type = "N"
  }
  
  attribute {
    name = "entryTime"
    type = "S"
  }
  
  # Global Secondary Index for plate lookups
  global_secondary_index {
    name               = "PlateIndex"
    hash_key           = "plate"
    projection_type    = "ALL"
  }
  
  # Global Secondary Index for parkingLot lookups
  global_secondary_index {
    name               = "ParkingLotIndex"
    hash_key           = "parkingLot"
    projection_type    = "ALL"
  }
  
  # Global Secondary Index for entryTime lookups
  global_secondary_index {
    name               = "EntryTimeIndex"
    hash_key           = "entryTime"
    projection_type    = "ALL"
  }
}

# IAM Role for Lambda functions
resource "aws_iam_role" "lambda_role" {
  name = "parking_lambda_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
      Effect = "Allow"
    }]
  })
}

# Attach IAM policies for Lambda to access DynamoDB and CloudWatch Logs
resource "aws_iam_role_policy_attachment" "lambda_dynamo_policy" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"
}

resource "aws_iam_role_policy_attachment" "lambda_log_policy" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Lambda function for Entry
resource "aws_lambda_function" "entry_handler" {
  function_name = "entryHandler"
  role          = aws_iam_role.lambda_role.arn
  runtime       = "provided.al2"
  architectures = ["arm64"]
  handler       = "bootstrap"
  filename      = "../cmd/lambda/entry-handler.zip"
  source_code_hash = filebase64sha256("../cmd/lambda/entry-handler.zip")

  environment {
    variables = {
      TABLE_NAME = aws_dynamodb_table.parking_tickets.name
    }
  }
}

# Lambda function for Exit
resource "aws_lambda_function" "exit_handler" {
  function_name = "exitHandler"
  role          = aws_iam_role.lambda_role.arn
  runtime       = "provided.al2"
  architectures = ["arm64"]
  handler       = "bootstrap"
  filename      = "../cmd/lambda/exit-handler.zip"
  source_code_hash = filebase64sha256("../cmd/lambda/exit-handler.zip")

  environment {
    variables = {
      TABLE_NAME = aws_dynamodb_table.parking_tickets.name
    }
  }
}

# API Gateway setup
resource "aws_api_gateway_rest_api" "parking_api" {
  name        = "Parking Lot API"
  description = "API for parking lot management system"
}

# Create API Gateway resources for entry and exit paths
resource "aws_api_gateway_resource" "entry_resource" {
  rest_api_id = aws_api_gateway_rest_api.parking_api.id
  parent_id   = aws_api_gateway_rest_api.parking_api.root_resource_id
  path_part   = "entry"
}

resource "aws_api_gateway_resource" "exit_resource" {
  rest_api_id = aws_api_gateway_rest_api.parking_api.id
  parent_id   = aws_api_gateway_rest_api.parking_api.root_resource_id
  path_part   = "exit"
}

# Create POST methods for each resource
resource "aws_api_gateway_method" "entry_method" {
  rest_api_id      = aws_api_gateway_rest_api.parking_api.id
  resource_id      = aws_api_gateway_resource.entry_resource.id
  http_method      = "POST"
  authorization    = "NONE"
  api_key_required = false

  request_parameters = {
    "method.request.querystring.plate"      = true
    "method.request.querystring.parkingLot" = true
  }
}

resource "aws_api_gateway_method" "exit_method" {
  rest_api_id      = aws_api_gateway_rest_api.parking_api.id
  resource_id      = aws_api_gateway_resource.exit_resource.id
  http_method      = "POST"
  authorization    = "NONE"
  api_key_required = false

  request_parameters = {
    "method.request.querystring.ticketId" = true
  }
}

# Add Lambda integrations
resource "aws_api_gateway_integration" "entry_integration" {
  rest_api_id             = aws_api_gateway_rest_api.parking_api.id
  resource_id             = aws_api_gateway_resource.entry_resource.id
  http_method             = aws_api_gateway_method.entry_method.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.entry_handler.invoke_arn
}

resource "aws_api_gateway_integration" "exit_integration" {
  rest_api_id             = aws_api_gateway_rest_api.parking_api.id
  resource_id             = aws_api_gateway_resource.exit_resource.id
  http_method             = aws_api_gateway_method.exit_method.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.exit_handler.invoke_arn
}

# Grant API Gateway permission to invoke the Lambda functions
resource "aws_lambda_permission" "api_gateway_entry_permission" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.entry_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.parking_api.execution_arn}/*/*/entry"
}

resource "aws_lambda_permission" "api_gateway_exit_permission" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.exit_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.parking_api.execution_arn}/*/*/exit"
}

# Create a deployment to make the API available
resource "aws_api_gateway_deployment" "api_deployment" {
  rest_api_id = aws_api_gateway_rest_api.parking_api.id
  description = "Initial deployment"

  depends_on = [
    aws_api_gateway_integration.entry_integration,
    aws_api_gateway_integration.exit_integration
  ]

  # Force redeployment when resources change
  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.entry_resource.id,
      aws_api_gateway_resource.exit_resource.id,
      aws_api_gateway_method.entry_method.id,
      aws_api_gateway_method.exit_method.id,
      aws_api_gateway_integration.entry_integration.id,
      aws_api_gateway_integration.exit_integration.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Create a stage for the deployment
resource "aws_api_gateway_stage" "prod_stage" {
  rest_api_id   = aws_api_gateway_rest_api.parking_api.id
  deployment_id = aws_api_gateway_deployment.api_deployment.id
  stage_name    = "prod"
} 