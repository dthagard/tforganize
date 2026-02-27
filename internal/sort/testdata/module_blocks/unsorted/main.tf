module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = "my-vpc"
  cidr = "10.0.0.0/16"

  depends_on = [module.network]
}

module "ec2" {
  source = "./modules/ec2"

  count = 2

  instance_type = "t3.micro"
  subnet_id     = module.vpc.private_subnets[0]
}
