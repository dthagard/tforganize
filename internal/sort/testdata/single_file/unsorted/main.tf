resource "tls_private_key" "proxmox" {
  algorithm = "ED25519"
}

resource "local_sensitive_file" "proxmox_private_key" {
  content         = tls_private_key.proxmox.private_key_openssh
  filename        = "${path.module}/${local.private_key}"
  file_permission = "0600"
}

resource "random_password" "proxmox" {
  length = 16
}

resource "proxmox_vm_qemu" "this" {
  for_each = { for node in local.nodes : node.name => node }

  name        = each.key
  desc        = each.key
  target_node = each.value.target_node

  clone = each.value.vm_template

  cores   = each.value.cores
  sockets = 1
  memory  = each.value.memory

  disk {
    type    = "scsi"
    storage = "local-lvm"
    size    = each.value.disk
  }

  network {
    model  = "virtio"
    bridge = "vmbr0"
  }

  ssh_user        = each.value.provisioner_user
  ssh_private_key = tls_private_key.proxmox.private_key_openssh
  sshkeys         = trim(chomp(tls_private_key.proxmox.public_key_openssh), " ")

  os_type      = "cloud-init"
  ipconfig0    = "gw=10.10.0.254,ip=${each.value.ip}/15"
  searchdomain = local.searchdomain
  nameserver   = local.nameserver

  ciuser     = each.value.provisioner_user
  cipassword = random_password.proxmox.result
  ci_wait    = 30 # How long in seconds to wait before provisioning

  onboot   = true # Whether to have the VM startup after the PVE node starts.
  oncreate = true # Whether to have the VM startup after the VM is created.
  tags     = "Terraform"

  # Options that conflict with other settings
  #bridge       = ""
  #disk_gb      = 0
  #mac          = ""
  #nic          = ""
  #storage      = ""
  #storage_type = ""

  # Wait until the instance is live before moving on to the next provisioner
  provisioner "remote-exec" {
    inline = [
      # We install python3 on amzn2 for two reasons:
      # 1. Ensure that python is available on the node for ansible
      # 2. Ensure the yum lock is removed before provisioning
      # Else, just check that the instance is live
      each.value.vm_template == local.vm_template_amazon2 ? "yum install -y python3" : "ip a"
    ]

    connection {
      type        = "ssh"
      user        = each.value.provisioner_user
      private_key = tls_private_key.proxmox.private_key_pem
      host        = each.value.ip
    }
  }

  provisioner "local-exec" {
    command = "ANSIBLE_FORCE_COLOR=True ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook -v -u ${each.value.provisioner_user} -i '${each.value.ip},' --private-key '${local.private_key}' -e 'service=${each.value.service}' --extra-vars '${each.value.extra_vars}' ../../ansible/${each.value.playbook}.yml"
  }

  # TODO figure out when to recycle the instance
  lifecycle {
    ignore_changes = all
  }
}
