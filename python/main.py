import time
import requests
import os


class Spider_baidu_image():
    def __init__(self):  # python类里面的构造函数
        self.image_urls = []  # 用于保存搜索到的图片url
        self.keyword = input("请输入搜索图片关键字:")  # 让用户输入搜索词
        # 让用户输入爬取数量，上限是多少，我们目前无法得知
        self.number = int(input("请输入搜索数量:"))
        # 拼接成我们的请求链接
        self.url = 'https://image.baidu.com/search/acjson'  # 百度图片的搜索接口
        self.headers = {
            # 设置UA，百度服务器后端通过该请求头来判断来访设备
            'User-Agent': 'Apipost client Runtime/+https://www.apipost.cn/',
        }
        self.pageCount = 30
        self.pn = 30
        self.params = ()
        cwd = os.getcwd()  # 字面意思，大概是获取系统当前目录
        self.dir = os.path.join(cwd, self.keyword)  # 把路径和文件夹名字合成一个路径

    def generate_params(self, page):
        pn = page*60
        rn = 60
        if self.pageCount == page:
            pn = self.number
            num = self.number%60
            if num != 0:
                rn = num
        return (
            ('tn', 'resultjson_com'),
            ('logid', '10982534902695910658'),
            ('ipn', 'rj'),
            ('ct', '201326592'),
            ('is', ''),
            ('fp', 'result'),
            ('fr', ''),
            ('word', self.keyword),
            ('cg', 'star'),
            ('queryWord',  self.keyword),
            ('cl', '2'),
            ('lm', '-1'),
            ('ie', 'utf-8'),
            ('oe', 'utf-8'),
            ('adpicid', ''),
            ('st', '-1'),
            ('z', ''),
            ('ic', '0'),
            ('hd', ''),
            ('latest', ''),
            ('copyright', ''),
            ('s', ''),
            ('se', ''),
            ('tab', ''),
            ('width', ''),
            ('height', ''),
            ('face', '0'),
            ('istype', '2'),
            ('qc', ''),
            ('nc', '1'),
            ('expermode', ''),
            ('nojc', ''),
            ('isAsync', ''),
            ('pn', pn),
            ('rn', rn),
            ('gsm', '3c'),
            (int(round(time.time() * 1000)), ''),
        )  # 请求所需的所有参数

    def mkdir_verify(self):  # 这是一个创建文件夹的函数
        if not os.path.exists(self.dir):  # 先检查一下同名文件夹存不存在，不存在就可以创建
            os.mkdir(self.dir)
        else:  # 如果存在了，那咱们就反馈一下，让用户设置一下新文件夹名字
            dirName = input("文件夹'{}'已存在，请输入新的文件夹名:".format(
                self.dir))  # 让用户输入文件夹名字
            # 当前路径和新设置的文件夹名字用/拼接，目的就是为了得到新的文件夹路径
            self.dir = os.path.join(os.getcwd(), dirName)
            self.mkdir_verify()  # 回调检查一下设置的文件名是否存在

    def get_img_url(self):  # 这是一个爬取搜索图片url的函数
        response = requests.get(
            self.url, headers=self.headers, params=self.params).json()  # 开始爬取搜索到的图片url
        json_data = response.get('data')  # 获取json的data字段数据，里面包含所需的图片url
        for i in json_data:  # 将这些数据遍历出来
            if i:
                # 图片url保存在这个thumbURL字段里
                self.image_urls.append(i.get('thumbURL'))

    def get_page_img_url(self):
        # 计算一下总共应该爬取多少页
        self.pageCount = int((self.number + 60 - 1) / 60)
        # 开始爬取每一页的图片url
        for page in range(self.pageCount):
            self.params = self.generate_params(page+1)
            self.get_img_url()

    def get_image(self):  # 爬取图片url的数据，下载保存在我们创建的文件夹里面
        saveNum = self.number
        urlsNum = len(self.image_urls)
        if self.number > urlsNum:
            answer = input("只爬取到{}张图片，没有达到{}张，是否下载？(y/n)".format(urlsNum, self.number))
            if answer == "n":
                return
            elif answer == "y":
                saveNum = urlsNum
            else:
                self.get_image()
                return

        for index in range(saveNum):  # 遍历我们得到的图片url数据
            with open(self.dir+'/{}.jpg'.format(index+1), 'wb') as f:  # 以wb模式新建并打开图片文件
                # 往打开的文件里写入爬取到的图片数据
                f.write(requests.get(self.image_urls[index], headers=self.headers).content)
                f.close()  # 打开的文件要记得关闭哦
        # 下载完一页，就提示一下
        print("{}张{}的图片已下载完成！保存位置{}".format(saveNum, self.keyword,self.dir))

    def __del__(self):  # 直接使用析构函数来处理
        self.mkdir_verify()  # 新建文件夹
        start = time.time()
        self.get_page_img_url()  # 爬取搜索到的图片url
        self.get_image()  # 下载图片数据
        print('耗时{}秒'.format(time.time()-start))

#入口
Spider_baidu_image()
