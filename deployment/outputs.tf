output "api_url" {
  value       = "${aws_api_gateway_rest_api.parking_api.id}.execute-api.${var.aws_region}.amazonaws.com/prod"
  description = "The URL of the API Gateway"
}

output "api_endpoint" {
  value       = aws_api_gateway_rest_api.parking_api.execution_arn
  description = "The execution ARN of the API Gateway"
}

output "entry_lambda_name" {
  value       = aws_lambda_function.entry_handler.function_name
  description = "The name of the entry Lambda function"
}

output "exit_lambda_name" {
  value       = aws_lambda_function.exit_handler.function_name
  description = "The name of the exit Lambda function"
}

output "dynamo_table_name" {
  value       = aws_dynamodb_table.parking_tickets.name
  description = "The name of the DynamoDB table"
} 