create table machine_info (
    _id int primary key auto_increment,
    hostname varchar(256),
    region varchar(64),
    image_id varchar(256),
    instance_type varchar(64),
    spot_price_limit float(5, 2),
    bandwith int(3),
    instance_id varchar(256),
    public_ip varchar(64),
    private_ip varchar(64),
    status  varchar(128),
);

insert into machine_info(hostname, region, image_id, instance_type,
    spot_price_limit, bandwith, instance_id, public_ip, private_ip,
    status) values('aly2-hn1-test-k8s-001', 'cn-beijing', 'centos_7_04_64_20G_alibase_201701015.vhd',
    'ecs.xn4.small', 0.8, 1, 'aly2-hn1-test-k8s-001', '', '172.18.49.126', 'machine-running');

alter table machine_info add create_time varchar(128);
