package result

type ResultVo[T any] struct {
	Code       int     `json:"code"`
	Successful bool    `json:"successful"`
	Msg        *string `json:"msg"`
	Data       *T      `json:"data"`
	Encrypt    bool    `json:"encrypt"`
}

type PageVo[T any] struct {
	PageNum  int   `json:"pageNum"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
	Pages    int64 `json:"pages"`
	List     any   `json:"list"`
}

func Ok(data any) ResultVo[any] {
	return ResultVo[any]{
		Code:       0,
		Successful: true,
		Data:       &data,
	}
}

func OkNull() ResultVo[any] {
	return ResultVo[any]{
		Code:       0,
		Successful: true,
	}
}

func Fail(code int, msg string) ResultVo[any] {
	return ResultVo[any]{
		Code:       code,
		Successful: code == 0,
		Msg:        &msg,
	}
}

// Success 判断是否成功
// @author: fallsea
// @return true 是
func (r ResultVo[T]) Success() bool {
	return r.Code == 0
}
