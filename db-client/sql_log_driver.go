package dbClient

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	log2 "github.com/wueasy/wueasy-go-tools/log"
)

// logDriver 包装原始 driver，拦截所有 SQL 执行并打印日志
type logDriver struct {
	driver.Driver
}

// Open 打开连接
func (d *logDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return &logConn{Conn: conn}, nil
}

// logConn 包装原始连接
type logConn struct {
	driver.Conn
}

// Prepare 准备语句
func (c *logConn) Prepare(query string) (driver.Stmt, error) {
	start := time.Now()
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		logSqlExec(context.Background(), query, nil, time.Since(start), err)
		return nil, err
	}
	return &logStmt{Stmt: stmt, query: query}, nil
}

// PrepareContext 准备语句（带context）
func (c *logConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	start := time.Now()
	var stmt driver.Stmt
	var err error

	if pc, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err = pc.PrepareContext(ctx, query)
	} else {
		stmt, err = c.Conn.Prepare(query)
	}

	if err != nil {
		logSqlExec(ctx, query, nil, time.Since(start), err)
		return nil, err
	}
	return &logStmt{Stmt: stmt, query: query}, nil
}

// ExecContext 执行语句（带context）
func (c *logConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	var result driver.Result
	var err error
	if ec, ok := c.Conn.(driver.ExecerContext); ok {
		result, err = ec.ExecContext(ctx, query, args)
	} else {
		return nil, driver.ErrSkip
	}
	logSqlExec(ctx, query, namedValuesToArgs(args), time.Since(start), err)
	return result, err
}

// QueryContext 查询语句（带context）
func (c *logConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	var rows driver.Rows
	var err error
	if qc, ok := c.Conn.(driver.QueryerContext); ok {
		rows, err = qc.QueryContext(ctx, query, args)
	} else {
		return nil, driver.ErrSkip
	}
	logSqlExec(ctx, query, namedValuesToArgs(args), time.Since(start), err)
	return rows, err
}

// BeginTx 开启事务
func (c *logConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if bc, ok := c.Conn.(driver.ConnBeginTx); ok {
		return bc.BeginTx(ctx, opts)
	}
	return c.Conn.Begin() //nolint
}

// logStmt 包装 Stmt，用于 Prepare 后的执行
type logStmt struct {
	driver.Stmt
	query string
}

func (s *logStmt) Exec(args []driver.Value) (driver.Result, error) {
	start := time.Now()
	result, err := s.Stmt.Exec(args)
	logSqlExec(context.Background(), s.query, valuesToArgs(args), time.Since(start), err)
	return result, err
}

// ExecContext 执行语句（带context）
func (s *logStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	var result driver.Result
	var err error
	if ec, ok := s.Stmt.(driver.StmtExecContext); ok {
		result, err = ec.ExecContext(ctx, args)
	} else {
		// 回退到 Exec
		values := make([]driver.Value, len(args))
		for i, a := range args {
			values[i] = a.Value
		}
		result, err = s.Stmt.Exec(values)
	}
	logSqlExec(ctx, s.query, namedValuesToArgs(args), time.Since(start), err)
	return result, err
}

func (s *logStmt) Query(args []driver.Value) (driver.Rows, error) {
	start := time.Now()
	rows, err := s.Stmt.Query(args)
	logSqlExec(context.Background(), s.query, valuesToArgs(args), time.Since(start), err)
	return rows, err
}

// QueryContext 查询语句（带context）
func (s *logStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	var rows driver.Rows
	var err error
	if qc, ok := s.Stmt.(driver.StmtQueryContext); ok {
		rows, err = qc.QueryContext(ctx, args)
	} else {
		// 回退到 Query
		values := make([]driver.Value, len(args))
		for i, a := range args {
			values[i] = a.Value
		}
		rows, err = s.Stmt.Query(values)
	}
	logSqlExec(ctx, s.query, namedValuesToArgs(args), time.Since(start), err)
	return rows, err
}

// logSqlExec 打印 SQL 日志
func logSqlExec(ctx context.Context, query string, args []interface{}, elapsed time.Duration, err error) {
	// driver.ErrSkip 是内部信号，不是真正的错误，忽略
	if err == driver.ErrSkip {
		return
	}
	ms := float64(elapsed.Nanoseconds()) / 1e6
	if err != nil {
		// 报错用 warn 级别输出（error 级别会附带完整调用栈，SQL 错误不需要）
		if len(args) > 0 {
			log2.Ctx(ctx).Warnf("[SQL] %.2fms | err=%v | %s | args=%v", ms, err, query, args)
		} else {
			log2.Ctx(ctx).Warnf("[SQL] %.2fms | err=%v | %s", ms, err, query)
		}
		return
	}
	// 正常 SQL 只在 debug 级别时输出
	if !log2.IsDebugEnabled() {
		return
	}
	if len(args) > 0 {
		log2.Ctx(ctx).Debugf("[SQL] %.2fms | %s | args=%v", ms, query, args)
	} else {
		log2.Ctx(ctx).Debugf("[SQL] %.2fms | %s", ms, query)
	}
}

// namedValuesToArgs 转换 NamedValue 到 []interface{}
func namedValuesToArgs(named []driver.NamedValue) []interface{} {
	args := make([]interface{}, len(named))
	for i, nv := range named {
		args[i] = nv.Value
	}
	return args
}

// valuesToArgs 转换 Value 到 []interface{}
func valuesToArgs(vals []driver.Value) []interface{} {
	args := make([]interface{}, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	return args
}

// registeredLogDrivers 记录已注册的日志 driver，避免重复注册
var registeredLogDrivers = map[string]bool{}

// logDriverName 返回带日志的 driver 名称
func logDriverName(driverName string) string {
	return fmt.Sprintf("%s-with-log", driverName)
}
