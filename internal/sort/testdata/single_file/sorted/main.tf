resource "local_sensitive_file" "proxmox_private_key" {
  content         = tls_private_key.proxmox.private_key_openssh
  file_permission = "0600"
  filename        = "${path.module}/${local.private_key}"
}

resource "proxmox_vm_qemu" "this" {
  for_each = { for node in local.nodes : node.name => node }

  ci_wait    = 30 # How long in seconds to wait before provisioning
  cipassword = random_password.proxmox.result
  ciuser     = each.value.provisioner_user
  clone      = each.value.vm_template
  cores      = each.value.cores
  desc       = each.key

  disk {
    size    = each.value.disk
    storage = "local-lvm"
    type    = "scsi"
  }

  ipconfig0  = "gw=10.10.0.254,ip=${each.value.ip}/15"
  memory     = each.value.memory
  name       = each.key
  nameserver = local.nameserver

  network {
    bridge = "vmbr0"
    model  = "virtio"
  }

  onboot          = true # Whether to have the VM startup after the PVE node starts.
  oncreate        = true # Whether to have the VM startup after the VM is created.
  os_type         = "cloud-init"
  searchdomain    = local.searchdomain
  sockets         = 1
  ssh_private_key = tls_private_key.proxmox.private_key_openssh
  ssh_user        = each.value.provisioner_user
  sshkeys         = trim(chomp(tls_private_key.proxmox.public_key_openssh), " ")
  tags            = "Terraform"
  target_node     = each.value.target_node

  # TODO figure out when to recycle the instance
  lifecycle {
    ignore_changes = all
  }

  provisioner "local-exec" {
    command = "ANSIBLE_FORCE_COLOR=True ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook -v -u ${each.value.provisioner_user} -i '${each.value.ip},' --private-key '${local.private_key}' -e 'service=${each.value.service}' --extra-vars '${each.value.extra_vars}' ../../ansible/${each.value.playbook}.yml"
  }

  # Options that conflict with other settings
  #bridge       = ""
  #disk_gb      = 0
  #mac          = ""
  #nic          = ""
  #storage      = ""
  #storage_type = ""

  # Wait until the instance is live before moving on to the next provisioner
  provisioner "remote-exec" {
    connection {
      host        = each.value.ip
      private_key = tls_private_key.proxmox.private_key_pem
      type        = "ssh"
      user        = each.value.provisioner_user
    }

    inline = [
      # We install python3 on amzn2 for two reasons:
      # 1. Ensure that python is available on the node for ansible
      # 2. Ensure the yum lock is removed before provisioning
      # Else, just check that the instance is live
      each.value.vm_template == local.vm_template_amazon2 ? "yum install -y python3" : "ip a"
    ]
  }
}

resource "random_password" "proxmox" {
  length = 16
}

resource "tls_private_key" "proxmox" {
  algorithm = "ED25519"
}
