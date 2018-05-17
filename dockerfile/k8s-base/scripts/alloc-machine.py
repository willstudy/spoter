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
from aliyunsdkecs.request.v20140526.StopInstanceRequest import StopInstanceRequest
from aliyunsdkecs.request.v20140526.AllocateEipAddressRequest import AllocateEipAddressRequest
from aliyunsdkecs.request.v20140526.AssociateEipAddressRequest import AssociateEipAddressRequest
from aliyunsdkecs.request.v20140526.DeleteInstanceRequest import DeleteInstanceRequest
from aliyunsdkecs.request.v20140526.ReleaseEipAddressRequest import ReleaseEipAddressRequest
from aliyunsdkecs.request.v20140526.UnassociateEipAddressRequest import UnassociateEipAddressRequest

MAX_RETRY = 3
# 停止实例后，多久触发删除
MAX_WAIT_S = 3
CREATE_ACTION = "create"
DELETE_ACTION = "delete"
STATUS_ACTION = "status"

logger = logging.getLogger("Alloc-ECS")
formatter = logging.Formatter('%(asctime)s %(funcName)s +%(lineno)d %(levelname)s: %(message)s')


file_handler = logging.FileHandler("alloc-machine.log")
file_handler.setFormatter(formatter)  # 可以通过setFormatter指定输出格式
"""
console_handler = logging.StreamHandler(sys.stdout)
console_handler.formatter = formatter
"""

logger.addHandler(file_handler)
logger.setLevel(logging.DEBUG)

class ECS_Operator:
    def __init__(self):
        self.accessKey = ""
        self.secretKey = ""
        self.region = "cn-beijing"
        self.imageID = "centos_7_04_64_20G_alibase_201701015.vhd"
        self.instanceType = ""
        self.groupID = ""
        self.price = ""
        self.keyName = ""
        self.bandwidth = ""
        self.action = ""
        self.instanceID = ""
        self.eip = ""
        self.assoID = ""
        self.vSwitchID= ""

    def set_AccessKey(self, accessKey):
        self.accessKey = accessKey

    def set_SecretKey(self, secretKey):
        self.secretKey = secretKey

    def set_Region(self, region):
        self.region = region

    def set_Region(self, region):
        self.region = region

    def set_ImageID(self, imageID):
        self.imageID = imageID

    def set_InstanceType(self, instanceType):
        self.instanceType = instanceType

    def set_GroupID(self, groupID):
        self.groupID = groupID

    def set_Price(self, price):
        self.price = price

    def set_KeyName(self, keyName):
        self.keyName = keyName

    def set_Bandwidth(self, bandwidth):
        self.bandwidth = bandwidth

    def set_Action(self, action):
        self.action = action

    def set_InstanceID(self, instanceID):
        self.instanceID = instanceID

    def set_EIP(self, eip):
        self.eip = eipID

    def set_AssoID(self, assoID):
        self.assoID = assoID

    def set_VSwitchID(self, vSwitchID):
        self.vSwitchID = vSwitchID

    def set_VSwitchID(self, vSwitchID):
        self.vSwitchID = vSwitchID

    def createECS_Client(self):
        return client.AcsClient(self.accessKey, self.secretKey, self.region)

    def do_action(self):
        if self.action == CREATE_ACTION:
            return self.createInstance()
        elif self.action == DELETE_ACTION:
            return self.deleteInstance()
        elif self.action == STATUS_ACTION:
            return self.getStatus()

    def getStatus(self):
        clt = self.createECS_Client()
        logger.debug("get instance: " + self.instanceID + " status.")
        response = self.getInstanceDetail(clt, self.instanceID)
        if response['code'] != 0:
            return response

        logger.info(response)

        ret = {}
        instances_info = response['msg'].get('Instances').get('Instance')
        if len(instances_info) < 1:
            ret['LockReason'] = ''
            ret['InstanceID'] = instanceID
            ret['ExpiredTime'] = ''
            ret['EipAddress'] = ''
            ret['Hostname'] = ''
            ret['InnerAddress'] = ''
            ret['msg'] = 'not found this instance'
            ret['code'] = 0
            return ret

        ret['LockReason'] = ''
        lock_reason = instances_info[0].get('OperationLocks').get('LockReason')
        if lock_reason is not None:
            for reason in lock_reason:
                if reason == "Recycling":
                    ret['LockReason'] = 'Recycling'
                    break

        ret['InstanceID'] = instanceID
        ret['ExpiredTime'] = instances_info[0].get('ExpiredTime')
        ret['EipAddress'] = instances_info[0].get('EipAddress').get('IpAddress')
        ret['Hostname'] = instances_info[0].get('HostName')
        ret['InnerAddress'] = instances_info[0].get('VpcAttributes').get('PrivateIpAddress').get('IpAddress')[0]
        ret['msg'] = "Create ECS successfully."
        ret['code'] = 0
        return response

    def deleteInstance(self):
        if self.instanceID != "":
            return self.deleteMachine()
        if self.eip != "":
            return self.deleteEIP()

    def deleteMachine(self):
        clt = self.createECS_Client()

        # 必须先停止 ecs，才可以删除
        request = StopInstanceRequest()
        request.set_InstanceId(self.instanceID)

        logger.debug("stop instance: " + self.instanceID)
        for retry in range(0, MAX_RETRY):
            resp = self._send_request(clt, request)
            if resp['code'] != 0 and resp['msg']['Code'] != 'IncorrectInstanceStatus':
                logger.warn("stop instance failed, due to: " + str(resp))
                continue
            else:
                break
        logger.debug("resp: " + str(resp))
        if resp['code'] != 0 and resp['msg']['Code'] != 'IncorrectInstanceStatus':
            return resp

        request = DeleteInstanceRequest()
        request.set_InstanceId(self.instanceID)
        logger.debug("delete instance: " + self.instanceID)

        while True:
            resp = self._send_request(clt, request)
            if resp['code'] == 0:
                logger.debug("delete instance OK, return %s" % resp['msg'])
                break

            logger.debug("response code: %s", str(response['code']))
            if resp['code'] != 'IncorrectInstanceStatus':
                logger.error("delete instance failed with %s" % resp['msg'])
                ret['code'] = 1
                ret['msg'] = "delete instance failed with" + str(resp['msg'])
                return ret
            logger.warn("delete instance failed with IncorrectInstanceStatus")
            time.sleep(1)
        logger.debug("delete instance done.")

        """删除 EIP
        request = UnassociateEipAddressRequest()
        request.set_InstanceId(self.instanceID)
        request.set_AllocationId(self.assoID)
        """


        ret = {}
        ret['code'] = 0
        ret['msg'] = "delete machine succ"
        ret['Hostname'] = self.instanceID
        ret['InstanceID'] = self.instanceID
        ret['InnerAddress'] = ""
        ret['EipAddress'] = ""
        ret['LockReason'] = ""
        ret['ExpiredTime'] = ""
        return ret

    def deleteEIP(self):
        clt = self.createECS_Client()
        request = ReleaseEipAddressRequest()
        request.set_AllocationId(self.assoID)
        return self._send_request(clt, request)

    def createInstance(self):
        ret = {}
        clt = self.createECS_Client()
        request = CreateInstanceRequest();
        request.set_ImageId(self.imageID)
        request.set_InstanceType(self.instanceType)
        request.set_SecurityGroupId(self.groupID)
        request.set_SpotPriceLimit(float(self.price))
        request.set_KeyPairName(self.keyName)
        request.set_InternetMaxBandwidthOut(int(self.bandwidth))
        request.set_IoOptimized('optimized')
        request.set_SystemDiskCategory('cloud_efficiency')
        request.set_InstanceChargeType('PostPaid')
        request.set_SpotStrategy('SpotWithPriceLimit')
        request.set_InternetChargeType('PayByTraffic')
        request.set_VSwitchId(self.vSwitchID)

        """步骤
        1. 创建 ECS
        2. 创建 EIP（暂时废弃）
        3. 启动 ECS
        4. 关联 EIP 到 ECS（暂时废弃）
        """

        response = self._send_request(clt, request)
        if response['code'] != 0:
            logger.warn("create instance failed with %s" % response['msg'])
            return response
        logger.info("create ecs done.")

        instanceID = response['msg'].get('InstanceId')
        associateID = ""

        """ 申请公网 IP
        if self.assoID != "" and self.eip != "":
            logger.debug("use existed eip to asso the ecs.")
            ret['EipAddress'] = self.eip
            associateID = self.assoID
        else:
            response = self.allocEIP(clt)
            if response['code'] != 0:
                logger.warn("alloc EIP failed with %s" % response['msg'])
                return response
            logger.info("alloc EIP response done.")
            ret['EipAddress'] = response['msg']['EipAddress']
            associateID = response['msg']['AllocationId']
        """

        while True:
            response = self.startInstance(clt, instanceID)
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

        """关联公网 IP 到 ECS 上
        while True:
            response = self.associateEIP(clt, associateID, instanceID)
            if response['code'] == 0:
                logger.info("associateEIP OK, response: %s" % response['msg'])
                break
            if response['msg']['Code'] != "IncorrectInstanceStatus":
                logger.error("associateEIP failed with %s" % response['msg'])
                return response
            logger.warn("start instance failed with IncorrectInstanceStatus")
            time.sleep(1)
        logger.info("associateEIP done.")
        """

        response = self.getInstanceDetail(clt, instanceID)
        logger.info(response)

        ret['LockReason'] = ''
        lock_reason = response['msg'].get('Instances').get('Instance')[0].get('OperationLocks').get('LockReason')
        if lock_reason is not None:
            for reason in lock_reason:
                if reason == "Recycling":
                    ret['LockReason'] = 'Recycling'
                    break

        ret['InstanceID'] = instanceID
        ret['ExpiredTime'] = response['msg'].get('Instances').get('Instance')[0].get('ExpiredTime')
        ret['EipAddress'] = response['msg'].get('Instances').get('Instance')[0].get('EipAddress').get('IpAddress')
        ret['Hostname'] = response['msg'].get('Instances').get('Instance')[0].get('HostName')
        ret['InnerAddress'] = response['msg'].get('Instances').get('Instance')[0].get('VpcAttributes').get('PrivateIpAddress').get('IpAddress')[0]
        ret['msg'] = "Create ECS successfully."
        ret['code'] = 0
        logger.info(ret)
        return ret

    def allocEIP(self, clt):
        request = AllocateEipAddressRequest()
        request.set_Bandwidth(1)
        request.set_InternetChargeType('PayByBandwidth')
        return self._send_request(clt, request)

    def associateEIP(self, clt, eipID, instanceID):
        request = AssociateEipAddressRequest()
        request.set_AllocationId(eipID)
        request.set_InstanceId(instanceID)
        return self._send_request(clt, request)

    def getInstanceDetail(self, clt, instanceID):
        ret = {}
        ret['code'] = 0
        request = DescribeInstancesRequest()
        request.set_InstanceIds(json.dumps([instanceID]))
        return self._send_request(clt, request)

    def _send_request(self, clt, request):
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

    def startInstance(self, clt, instanceID):
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
    -b or --bandwidth: ECS 的带宽参数
    -v or --vSwitchID: ECS 虚拟专用网的 ID
    --action: 执行的动作
    --instanceID: ECS 的 ID 信息
    --eipID: eip 的 ID 信息
    --assoID: ECS 关联 IP 的信息
    --vSwitchID: ECS的交换机ID信息
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
    bandwidth = ""
    action = ""
    instanceID = ""
    eipID = ""
    assoID = ""
    vSwitchID = ""

    try:
        opts, args = getopt.getopt(sys.argv[1:], "x:a:s:r:i:t:g:p:k:h:b:v", ["accessKey=",
        "secretKey=", "region=", "imageID=", "instanceType=", "groupID=", "price=",
        "keyName=", "bandwidth=", "action=", "instanceID=", "eipID=", "assoID=", "vSwitchID=", "help"])

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
            elif opt in ("-b", "--bandwidth"):
                bandwidth = arg
            elif opt in ("-v", "--vSwitchID"):
                vSwitchID = arg
            elif opt in ("xxx", "--action"):
                action = arg
            elif opt in ("xxx", "--instanceID"):
                instanceID = arg
            elif opt in ("xxx", "--eipID"):
                eipID = arg
            elif opt in ("xxx", "--assoID"):
                assoID = arg
            elif opt in ("xxx", "--vSwitchID"):
                vSwitchID = arg
            elif opt in ("-h", "--help"):
                usage()
                sys.exit(0)
    except getopt.GetoptError:
        msg = "alloc-machine.py -a <accessKey> -s <secretKey> -r <region>"
        msg += " -i <imageID> -t <instanceType> -g <groupID>"
        output['code'] = 2
        output['msg'] = msg

    if action == "":
        output['code'] = 1
        output['msg'] = "action can not be NULL."
    if accessKey == "":
        output['code'] = 1
        output['msg'] = "accessKey can not be NULL."
    if secretKey == "":
        output['code'] = 1
        output['msg'] = "secretKey can not be NULL."
    if region == "":
        output['code'] = 1
        output['msg'] = "region can not be NULL."

    if action == CREATE_ACTION:
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
        if bandwidth == "":
            output['code'] = 1
            output['msg'] = "bandwidth can not be NULL."
        if vSwitchID == "":
            output['code'] = 1
            output['msg'] = "vSwitchID can not be NULL."

    if 'code' in output and output['code'] != 0:
        logger.warn(output['msg'])
        sys.exit(output['code'])

    ep = ECS_Operator()
    ep.set_AccessKey(accessKey)
    ep.set_SecretKey(secretKey)
    ep.set_Region(region)
    ep.set_ImageID(imageID)
    ep.set_InstanceType(instanceType)
    ep.set_GroupID(groupID)
    ep.set_Price(price)
    ep.set_KeyName(keyName)
    ep.set_Bandwidth(bandwidth)
    ep.set_Action(action)
    ep.set_InstanceID(instanceID)
    ep.set_EIP(eipID)
    ep.set_AssoID(assoID)
    ep.set_VSwitchID(vSwitchID)

    ret = ep.do_action()
    logger.debug(ret)

    if ret['code'] != 0:
        sys.exit(ret['code'])

    result = '{'
    if 'code' in ret:
        result = result + '"code": ' + str(ret['code'])
    if 'msg' in ret:
        result = result + ', "msg": "' + str(ret['msg']) + '"'
    if 'EipAddress' in ret:
        result = result + ', "EipAddress": "' + str(ret['EipAddress']) + '"'
    if 'InnerAddress' in ret:
        result = result + ', "InnerAddress": "' + str(ret['InnerAddress']) + '"'
    if 'Hostname' in ret:
        result = result + ', "Hostname": "' + str(ret['Hostname']) + '"'
    if 'InstanceID' in ret:
        result = result + ', "InstanceID": "' + str(ret['InstanceID']) + '"'
    if 'ExpiredTime' in ret:
        result = result + ', "ExpiredTime": "' + str(ret['ExpiredTime']) + '"'
    if 'LockReason' in ret:
        result = result + ', "LockReason": "' + str(ret['LockReason']) + '"'
    result = result + '}'

    logger.debug("result: %s", result)
    print result
