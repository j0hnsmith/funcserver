# funcserver

[![GoDoc](https://godoc.org/github.com/j0hnsmith/funcserver?status.svg)](https://godoc.org/github.com/j0hnsmith/funcserver)
[![Go Report Card](https://goreportcard.com/badge/github.com/j0hnsmith/funcserver)](https://goreportcard.com/report/github.com/j0hnsmith/funcserver)

This project provides conversion wrappers to make a `http.Handler`, such as a router, work with function-as-a-service (faas) providers (currently only AWS Lambda via an ALB implemented).

## Why?
faas means you don't have to keep the server running - no monitoring, upgrades, patching etc etc.  

### Caveats
Of course sending requests to a `http.Handler` without the usual server will have some caveats

* no streaming requests/responses
* request/response body size restrictions (1mb for lambda via ALB)
* faas suffers from slow starts ([AWS VPC very slow starts to be solved in 2019](https://www.nuweba.com/AWS-Lambda-in-a-VPC-will-soon-be-faster))

Private VPC aside, whether this is a problem for you depends on your use case, it shouldn't be for the majority.

## ALB + Lambda example
This example uses a gorilla mux router.

```go

package main

import (
	"fmt"
    "net/http"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/gorilla/mux"
    "github.com/j0hnsmith/funcserver/alblambda"
)

func main() {
    router := mux.NewRouter()
	links := `<a href="/">Home</a><br/><a href="/products">Products</a><br/><a href="/articles">Articles</a><br/>`

	router.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {resp.Write([]byte(fmt.Sprintf("<h1>Home</h1>%s", links)))})
	router.HandleFunc("/products", func(resp http.ResponseWriter, req *http.Request) {resp.Write([]byte(fmt.Sprintf("<h1>Products</h1>%s", links)))})
	router.HandleFunc("/articles", func(resp http.ResponseWriter, req *http.Request) {resp.Write([]byte(fmt.Sprintf("<h1>Articles</h1>%s", links)))})

    // wrap handler to automatically convert requests/responses
    lambda.Start(alblambda.WrapHTTPHandler(router, alblambda.ResponseOptions{}))
}
```

## AWS ALB+Lambda working example

You can try it out for yourself (in as little as a few minutes if you've got terraform and have an AWS account configured), here's some example terraform config to run the example, to use it...

* `make build` to build and package the code into a zip for deployment to lambda
* save below in file `main.tf`
* modify `region_vpc`, `zip_path` & `region` values
* run `terraform init` to install the provider etc
* run `terraform apply` with [AWS provider](https://www.terraform.io/docs/providers/aws/) configured
* _with devtools open on the network tab to get an idea of the cold start delay (optional)_, visit the loadbalancer url output via `alb_dns_name`
* also visit `/products` & `/articles` endpoints


```terraform
locals {
  region              = "eu-west-2"
  region_vpc          = "vpc-your-id"
  function_name       = "funcserver_test"
  zip_path            = "/absolute/path/to/src/github.com/j0hnsmith/funcserver/artifacts/main.zip"
}

provider "aws" {
  region  = "${local.region}"
  version = "= 1.53"
}

data "aws_vpc" "eu_west_2" {
  id = "${local.region_vpc}"
}

data "aws_iam_policy_document" "lambda_default" {
  statement {
    // default lambda stuff
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = [
      "*",
    ]
  }
}

resource "aws_iam_role_policy" "funcserver_test" {
  name = "${local.function_name}_policy"
  role = "${aws_iam_role.lambda_default.name}"

  policy = "${data.aws_iam_policy_document.lambda_default.json}"

  provisioner "local-exec" {
    command = "sleep 10"
  }
}

resource "aws_iam_role" "lambda_default" {
  name        = "${local.function_name}_iam_role"
  description = "allow assume role to lambda"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_lambda_function" "function_definition" {
  filename         = "${local.zip_path}"
  function_name    = "${local.function_name}"
  source_code_hash = "${base64sha256(file("${local.zip_path}"))}"
  role             = "${aws_iam_role.lambda_default.arn}"
  description      = "funcserver test lambda function, called via an alb"
  handler          = "main"
  runtime          = "go1.x"
  timeout          = "10"
  memory_size      = "128"

  depends_on = ["aws_iam_role_policy.funcserver_test"]
}

data "aws_subnet_ids" "region" {
  vpc_id = "${local.region_vpc}"
}

data "aws_subnet" "example" {
  count = "${length(data.aws_subnet_ids.region.ids)}"
  id    = "${data.aws_subnet_ids.region.ids[count.index]}"
}

data "aws_security_group" "region_default" {
  vpc_id = "${local.region_vpc}"
  name   = "default"
}

resource "aws_lb" "funcserver_test" {
  name               = "funcserver-test"
  load_balancer_type = "application"
  internal           = false

  security_groups = ["${aws_security_group.alb_public.id}"]
  subnets         = ["${data.aws_subnet_ids.region.ids}"]

  tags {
    created-by = "terraform"
  }
}

output "alb_dns_name" {
  value       = "${aws_lb.funcserver_test.dns_name}"
  description = "Output the dns name of the internal load balancer for DNS records"
}

resource "aws_alb_listener" "funcserver_test_http" {
  load_balancer_arn = "${aws_lb.funcserver_test.arn}"
  port              = "80"
  protocol          = "HTTP"

  default_action {
    target_group_arn = "${aws_lb_target_group.funcserver_test.arn}"
    type             = "forward"
  }
}

resource "aws_lb_target_group" "funcserver_test" {
  name        = "tf-example-lb-tg"
  target_type = "lambda"

  //  health_check {
  //    interval = 30
  //    path     = "/"
  //    timeout  = 15
  //    matcher  = "200"
  //  }

  // this is to create the new default target group used by the default group before deleting it?
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "alb_public" {
  name        = "alb-public-test"
  description = "Security group allowing traffic from http 80 to the terraform test public ALB"
  vpc_id      = "${local.region_vpc}"

  tags {
    Created-by = "terraform"
  }

  // we are using the ephemeral port range for containers
  ingress {
    from_port = 80
    to_port   = 80
    protocol  = "tcp"

    cidr_blocks = ["0.0.0.0/0"]
  }

  //  ingress {
  //    from_port = 443
  //    to_port   = 443
  //    protocol  = "tcp"
  //
  //    cidr_blocks = ["0.0.0.0/0"]
  //  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lambda_permission" "allow_call_from_alb" {
  statement_id  = "AllowExecutionFromALB"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.function_definition.function_name}"
  principal     = "elasticloadbalancing.amazonaws.com"
  source_arn    = "${aws_lb_target_group.funcserver_test.arn}"
}

resource "aws_lb_target_group_attachment" "lambda_func" {
  target_group_arn = "${aws_lb_target_group.funcserver_test.arn}"
  target_id        = "${aws_lambda_function.function_definition.arn}"
}
````
