const { useState, useMemo } = React;
const validPeriods = {
    "10m": "10 минут",
    "30m": "30 минут",
    "60m": "1 час",
    "120m": "2 часа",
    "180m": "3 часа",
    "240m": "4 часа",
    "480m": "8 часов",
}
const App = () => {
    const [validFor, setValidFor] = useState("60m")
    const [text, setText] = useState('');
    const [errMsg, setErrMsg] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [token, setToken] = useState('');
    const [isCoped, setIsCoped] = useState(false);
    const [passwordLen, setPasswordLen] = useState(8);
    const [settings, setSettings] = useState(false);
    

    const handleTemplate = (template) => {
        switch (template) {
            case 'lp':
                setText('login: \npassword:')
                return
            case 'lpg':
                $.get({
                    url: `/api/password?len=${passwordLen}` ,
                    success: function (response) {
                        setText('login: \npassword: ' + response)
                    },
                    error: function (error) {
                        console.log(error.responseText);
                        setText('login: \npassword:')
                    }
                })
                return
            default:
        }
    }
    const handleSettings= () => {
        setSettings(!settings)
    }
    const handleValidTimeChange = (validTime) => {
        setValidFor(validTime)
    }
    const validForString = useMemo(() => {
        return validPeriods[validFor]
    }, [validFor])

    const sendingData = useMemo(() => {
        return JSON.stringify({ text: text, ttl: validFor })
    }, [validFor, text])

    const onetimeLink = useMemo(() => {
        if (token === '') return ''
        return `${window.location.href.replace('#', '')}token.html?token=${token}`
    }, [token])

    const isReadyLink = useMemo(() => { return onetimeLink === '' ? false : true }, [onetimeLink])

    const handleClick = () => {
        if (isLoading) return
        setToken('')
        setErrMsg('')
        setIsCoped(false)
        setIsLoading(true)
        $.post({
            url: "/api/token",
            data: sendingData,
            dataType: "json",
            success: function (response) {
                setToken(response)
                setIsLoading(false)
            },
            error: function (error) {
                console.log(error.responseText);
                setErrMsg(error.responseJSON.message)
                setToken('')
                setIsLoading(false)
            }
        })
    }

    //    console.log('href', window.location.href.replace('#', ''))
    return (
        <div>
            <div className="container" style={{ marginTop: "30px" }}>
                <div className="col-xs-10 col-xs-offset-1 jumbotron pb-4 pt-4 mb-2">
                    <p className="lead">Создайте ссылку на выше сообщение, которая сработает только однажды.</p>

                    <div className="input-group input-group-sm mb-0">
                        <textarea autofocus="" className="form-control bg-dark text-white " style={{ fontSize: "1rem" }} value={text} aria-label="With textarea" rows="4" onChange={(event) => { setText(event.target.value) }}></textarea>
                    </div>

                    <p className="text-muted" style={{ fontSize: "0.8rem" }}>Срок действия: {validForString}</p>
                    <div class="btn-group" role="group">
                        <button id="btnGroupDrop1" type="button" class="btn btn-secondary dropdown-toggle btn-sm" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false">
                            Срок действия
                        </button>
                        <div class="dropdown-menu" aria-labelledby="btnGroupDrop1">
                            <a className="dropdown-item" href="#" onClick={() => handleValidTimeChange('10m')} role="button">10 минут</a>
                            <a className="dropdown-item" href="#" onClick={() => handleValidTimeChange('30m')} role="button">30 минут</a>
                            <a className="dropdown-item" href="#" onClick={() => handleValidTimeChange('60m')} role="button">1 час</a>
                            <a className="dropdown-item" href="#" onClick={() => handleValidTimeChange('120m')} role="button">2 часа</a>
                            <a className="dropdown-item" href="#" onClick={() => handleValidTimeChange('180m')} role="button">3 часа</a>
                            <a className="dropdown-item" href="#" onClick={() => handleValidTimeChange('240m')} role="button">4 часа</a>
                            <a className="dropdown-item" href="#" onClick={() => handleValidTimeChange('480m')} role="button">8 часов</a>
                        </div>
                    </div>
                    <div class="btn-group ml-4" role="group">
                        <button id="btnGroupDrop8" type="button" class="btn btn-secondary dropdown-toggle btn-sm" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false">
                            Шаблоны сообщений
                        </button>
                        <div class="dropdown-menu" aria-labelledby="btnGroupDrop8">
                            <a className="dropdown-item" href="#" onClick={() => handleTemplate('lp')} role="button">Логин, пароль</a>
                            <a className="dropdown-item" href="#" onClick={() => handleTemplate('lpg')} role="button">Логин, пароль (сгенерированный)</a>
                        </div>

                    </div>
                    <div class="btn-group ml-4" role="group">
                        <div class="form-outline">
                            <button type="button" class="btn btn-secondary btn-sm" onClick={() => handleSettings()}>Настройки</button>
                        </div>
                    </div>

                    {settings &&<div class="form-row">

                        <div class="col-md-2 mt-2">
                            <small >длина пароля</small>
                            <input value={passwordLen} onChange={(e) => setPasswordLen(e.target.value)} min="6" max="20" type="number" id="typeNumber" class="form-control form-control-sm" />
                        </div>
                    </div>}
                    <hr className="my-4" />

                    <div className="container" style={{ marginTop: "20px" }}>
                        <div className="row ">
                            <div className="col-3">
                                <a className="btn btn-primary  " href="#" onClick={handleClick} role="button">Создать</a>
                            </div>

                        </div>
                    </div>

                    {isLoading && <div class="d-flex justify-content-center mt-4">
                        <div class="spinner-border text-success" style={{ width: '3rem', height: '3rem' }} role="status">
                            <span class="sr-only">Loading...</span>
                        </div>
                    </div>}
                    {!isLoading && isReadyLink && <div class="input-group" style={{ marginTop: '70px' }}>
                        <input type="text" value={onetimeLink} className={isCoped ? "form-control is-valid" : "form-control"} placeholder="Одноразовая ссылка" aria-label="Одноразовая ссылка" aria-describedby="button-addon2" />
                        <div class="input-group-append">
                            <CopyToClipboard onCopy={() => setIsCoped(true)} text={onetimeLink}>
                                <button class="btn btn-outline-secondary" type="button" id="button-addon2" >Копировать в Clipboard</button>
                            </CopyToClipboard>
                        </div>

                    </div>}
                    {errMsg &&
                        <div class="alert alert-danger mt-4" role="alert">
                            {errMsg}
                        </div>
                    }
                </div>
            </div >
            <div className="container" style={{ marginTop: "30px" }}>
                <div className="col-xs-10 col-xs-offset-2 pt-2 mb-2 ">
                    <div class="row no-gutters justify-content-md-center mb-4 p-0">
                        <div class="pb-2 pt-2 col-md-1 align-self-center  text-white font-weight-bold" align="center" style={{ fontSize: "2.4rem", backgroundColor: "#ddd" }}>
                            1
                        </div>
                        <div class="col-md-7 bg-light text-dark">
                            <div class="p-2 pl-4">
                                <h6 class="card-title text-muted ">Напишите сообщение</h6>
                                <p class="card-text"><small class="text-muted">Напишите секретное сообщение, которое будет зашифровано</small></p>
                            </div>
                        </div>
                    </div>
                    <div class="row no-gutters justify-content-md-center mb-4">
                        <div class="pb-2 pt-2 col-md-1 align-self-center text-white font-weight-bold" align="center" style={{ fontSize: "2.4rem", backgroundColor: "#ddd" }}>
                            2
                        </div>
                        <div class="col-md-7 bg-light text-dark">
                            <div class="p-2 pl-4">
                                <h6 class="card-title text-muted ">Выберите срок действия ссылки</h6>
                                <p class="card-text"><small class="text-muted">После истечения установленного срока ссылку открыть будет невозможно</small></p>
                            </div>
                        </div>
                    </div>
                    <div class="row no-gutters justify-content-md-center mb-4">
                        <div class="pb-2 pt-2 col-md-1 align-self-center text-white font-weight-bold" align="center" style={{ fontSize: "2.4rem", backgroundColor: "#ddd" }}>
                            3
                        </div>
                        <div class="col-md-7 bg-light text-dark">
                            <div class="p-2 pl-4">
                                <h6 class="card-title text-muted ">Создайте и скопируйте ссылку</h6>
                                <p class="card-text"><small class="text-muted">Нажмите кнопку 'Создать' и затем скопируйте созданную ссылку</small></p>
                            </div>
                        </div>
                    </div>
                    <div class="row no-gutters justify-content-md-center">
                        <div class="pb-2 pt-2 col-md-1 align-self-center text-white font-weight-bold" align="center" style={{ fontSize: "2.4rem", backgroundColor: "#ddd" }}>
                            4
                        </div>
                        <div class="col-md-7 bg-light text-dark">
                            <div class="p-2 pl-4">
                                <h6 class="card-title text-muted ">Отправьте ссылку любым способом</h6>
                                <p class="card-text"><small class="text-muted">Отправьте ссылку удобным для вас способом, ссылку можно будет открыть только один раз</small></p>
                            </div>
                        </div>
                    </div>


                </div>
            </div>
        </div >
    )
}

ReactDOM.render(<App />, document.getElementById('app'));

