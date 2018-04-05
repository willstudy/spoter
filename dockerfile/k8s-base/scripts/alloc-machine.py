#  coding=utf-8
import json
import sys
import logging
import time
import getopt
from aliyunsdkcore import client
from aliyunsdkecs.request.v20140526.CreateInstanceRequest import CreateInstanceRequest
from aliyunsdkecs.request.v20140526.DescribeInstancesRequest import DescribeInstancesRequest
from aliyunsdkecs.request.v20140526.StartInstanceRequest import StartInstanceRequest
from aliyunsdkecs.request.v20140526.AllocateEipAddressRequest import AllocateEipAddressRequest
from aliyunsdkecs.request.v20140526.AssociateEipAddressRequest import AssociateEipAddressRequest

MAX_RETRY = 3
logger = logging.getLogger("Alloc-ECS")
formatter = logging.Formatter('%(asctime)s %(funcName)s +%(lineno)d %(levelname)s: %(message)s')

"""
file_handler = logging.FileHandler(".log")
file_handler.setFormatter(formatter)  # 可以通过setFormatter指定输出格式
"""

console_handler = logging.StreamHandler(sys.stdout)
console_handler.formatter = formatter

logger.addHandler(console_handler)
logger.setLevel(logging.DEBUG)

def createECS(accessKey, secretKey, region, imageID, instanceType, groupID, price, keyName):
    clt = client.AcsClient(accessKey, secretKey, region)
    return createInstance(clt, imageID, instanceType, groupID, price, keyName)

def createInstance(clt, imageID, instanceType, groupID, price, keyName):
    ret = {}
    request = CreateInstanceRequest();
    request.set_ImageId(imageID)
    request.set_InstanceType(instanceType)
    request.set_SecurityGroupId(groupID)
    request.set_SpotPriceLimit(float(price))
    request.set_KeyPairName(keyName)
    request.set_IoOptimized('optimized')
    request.set_SystemDiskCategory('cloud_efficiency')
    request.set_InstanceChargeType('PostPaid')
    request.set_SpotStrategy('SpotWithPriceLimit')
    request.set_InternetChargeType('PayByBandwidth')
    request.set_InternetMaxBandwidthOut(1)

    response = _send_request(request, clt)
    if response['code'] != 0:
        logger.warn("create instance failed with %s" % response['msg'])
        return response

    instanceID = response['msg'].get('InstanceId')

    response = allocEIP(clt)
    if response['code'] != 0:
        logger.warn("alloc EIP failed with %s" % response['msg'])
        return response
    logger.info("alloc EIP response: %s" % response['msg'])
    ret['EipAddress'] = response['msg']['EipAddress']
    associateID = response['msg']['AllocationId']

    while True:
        response = startInstance(clt, instanceID)
        if response['code'] == 0:
            logger.debug("start instance OK, return %s" % response['msg'])
            break
        logger.debug("response code: %s", str(response['code']))
        if response['code'] != 'IncorrectInstanceStatus':
            logger.error("start instance failed with %s" % response['msg'])
            ret['code'] = 1
            ret['msg'] = "start instance failed with" + str(response['msg'])
            return ret
        logger.warn("start instance failed with IncorrectInstanceStatus")
        time.sleep(1)
    logger.debug("start instance done.")

    response = associateEIP(clt, associateID, instanceID)
    if response['code'] != 0:
        logger.warn("associateEIP failed with %s" % response['msg'])
        return response
    logger.info("associateEIP response: %s" % response['msg'])

    response = getInstanceDetail(clt, instanceID)
    logger.info(response)
    ret['Hostname'] = response['msg'].get('Instances').get('Instance')[0].get('HostName')
    ret['msg'] = "Create ECS successfully."
    ret['code'] = 0
    logger.info(ret)
    return ret

def startInstance(clt, instanceID):
    ret = {}
    request = StartInstanceRequest()
    request.set_InstanceId(instanceID)
    request.set_accept_format('json')
    try:
        response_str = json.loads(clt.do_action(request))
        logger.debug("response: %s" % response_str)
        if 'Code' in response_str:
            ret['code'] = response_str['Code']
        else:
            ret['code'] = 0
        ret['msg'] = response_str
    except Exception as e:
        ret['code'] = 1
        ret['msg'] = str(e)
    return ret

def allocEIP(clt):
    request = AllocateEipAddressRequest()
    request.set_Bandwidth(1)
    request.set_InternetChargeType('PayByBandwidth')
    return _send_request(request, clt)

def associateEIP(clt, eipID, instanceID):
    request = AssociateEipAddressRequest()
    request.set_AllocationId(eipID)
    request.set_InstanceId(instanceID)
    return _send_request(request, clt)

def _send_request(request, clt):
    ret = {}
    request.set_accept_format('json')
    try:
        response_str = clt.do_action(request)
        response_detail = json.loads(response_str)
        logger.debug("response: %s" % response_detail)
        if 'Code' in response_detail:
            ret['code'] = response_detail['Code']
        else:
            ret['code'] = 0
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
    return _send_request(request, clt)

def usage():
    logger.info("""
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
    """)

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
        logger.warn(output['msg'])
        sys.exit(output['code'])

    ret = createECS(accessKey, secretKey, region, imageID, instanceType, groupID, price, keyName)
    logger.debug(ret['msg'])
    sys.exit(ret['code'])
