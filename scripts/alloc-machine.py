#  coding=utf-8
import json
import sys
import getopt
from aliyunsdkcore import client
from aliyunsdkecs.request.v20140526.CreateInstanceRequest import CreateInstanceRequest
from aliyunsdkecs.request.v20140526.DescribeInstancesRequest import DescribeInstancesRequest
from aliyunsdkecs.request.v20140526.StartInstanceRequest import StartInstanceRequest

MAX_RETRY = 3

def createECS(accessKey, secretKey, region, imageID, instanceType, groupID, price, keyName):
    ret = {}
    ret['code'] = 0
    ret['msg'] = 'Create ECS successfully.'
    clt = client.AcsClient(accessKey, secretKey, region)
    response = createInstance(clt, imageID, instanceType, groupID, price, keyName)
    if response['code'] != 0:
        ret['code'] = response['code']
        ret['msg'] = response['msg']
    return ret

def createInstance(clt, imageID, instanceType, groupID, price, keyName):
    request = CreateInstanceRequest();
    request.set_ImageId(imageID)
    request.set_InstanceType(instanceType)
    request.set_SecurityGroupId(groupID)
    request.set_SpotPriceLimit(float(price))
    request.set_KeyPairName(keyName)
    request.set_IoOptimized('optimized')
    request.set_SystemDiskCategory('cloud_ssd')
    request.set_InstanceChargeType('PostPaid')
    request.set_SpotStrategy('SpotWithPriceLimit')
    request.set_InternetChargeType('PayByBandwidth')
    request.set_InternetMaxBandwidthOut(100)
    request.set_InternetMaxBandwidthIn(100)

    response = _send_request(request, clt)
    if response['code'] != 0:
        return response
    print response
    instanceID = response['msg'].get('InstanceId')
    retry = 0
    ret = {}
    while retry < MAX_RETRY:
        ret = startInstance(clt, instanceID)
        if ret['code'] == 0:
            break
        retry = retry + 1
    if retry >= MAX_RETRY:
        return ret

    return getInstanceDetail(clt, instanceID)

def startInstance(clt, instanceID):
    ret = {}
    ret['code'] = 0
    ret['msg'] = 'Start instance successfully.'
    request = StartInstanceRequest()
    request.set_InstanceId(instanceID)
    try:
        response_str = clt.do_action(request)
        ret['msg'] = response_str
    except Exception as e:
        ret['code'] = 1
        ret['msg'] = str(e)
    return ret

def _send_request(request, clt):
    ret = {}
    ret['code'] = 0
    ret['msg'] = "Send request successfully."
    request.set_accept_format('json')
    try:
        response_str = clt.do_action(request)
        response_detail = json.loads(response_str)
        ret['msg'] = response_detail
    except Exception as e:
        ret['code'] = 1
        ret['msg'] = str(e)
    return ret

def getInstanceDetail(clt, instanceID):
    ret = {}
    ret['code'] = 0
    request = DescribeInstancesRequest()
    request.set_InstanceIds(json.dumps([instanceID]))
    response = _send_request(request, clt)
    print response
    ret['msg'] = response
    return ret


def usage():
    print """
    Usage:sys.args[0] [option]
    -h or --help：显示帮助信息
    -a or --accessKey: 阿里云的 accessKey
    -s or --secretKey: 阿里云的 secretKey
    -r or --region: region 信息
    -i or --imageID: ECS 的镜像信息
    -t or --instanceType: ECS 的型号规格
    -g or --groupID: 阿里云的安全组
    -p or --price: 后付费的价格
    -k or --keyName: SSH 信任的秘钥名
    """

if __name__ == '__main__':
    output = {}

    accessKey = ""
    secretKey = ""
    region = "cn-beijing"
    imageID = "centos_7_04_64_20G_alibase_201701015.vhd"
    instanceType = ""
    groupID = ""
    price = ""
    keyName = ""

    try:
        opts, args = getopt.getopt(sys.argv[1:], "a:s:r:i:t:g:p:k:h",["accessKey=","secretKey=",
        "region=", "imageID=", "instanceType=", "groupID=", "price=", "keyName=", "help"])

        for opt, arg in opts:
            if opt in ("-a", "--accessKey"):
                accessKey = arg
            elif opt in ("-s", "--secretKey"):
                secretKey = arg
            elif opt in ("-r", "--region"):
                region = arg
            elif opt in ("-i", "--imageID"):
                imageID = arg
            elif opt in ("-t", "--instanceType"):
                instanceType = arg
            elif opt in ("-g", "--groupID"):
                groupID = arg
            elif opt in ("-p", "--price"):
                price = arg
            elif opt in ("-k", "--keyName"):
                keyName = arg
            elif opt in ("-h", "--help"):
                usage()
                sys.exit(0)
    except getopt.GetoptError:
        msg = "alloc-machine.py -a <accessKey> -s <secretKey> -r <region>"
        msg += " -i <imageID> -t <instanceType> -g <groupID>"
        output['code'] = 2
        output['msg'] = msg

    if accessKey == "":
        output['code'] = 1
        output['msg'] = "accessKey can not be NULL."
    if secretKey == "":
        output['code'] = 1
        output['msg'] = "secretKey can not be NULL."
    if region == "":
        output['code'] = 1
        output['msg'] = "region can not be NULL."
    if imageID == "":
        output['code'] = 1
        output['msg'] = "imageID can not be NULL."
    if instanceType == "":
        output['code'] = 1
        output['msg'] = "instanceType can not be NULL."
    if groupID == "":
        output['code'] = 1
        output['msg'] = "groupID can not be NULL."
    if price == "":
        output['code'] = 1
        output['msg'] = "price can not be NULL."
    if keyName == "":
        output['code'] = 1
        output['msg'] = "keyName can not be NULL."

    if 'code' in output and output['code'] != 0:
        print output['msg']
        sys.exit(output['code'])

    ret = createECS(accessKey, secretKey, region, imageID, instanceType, groupID, price, keyName)
    print ret['msg']
    sys.exit(ret['code'])
