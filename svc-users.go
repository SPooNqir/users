package users

import (
	"context"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	lib "gitlab.com/SpoonQIR/Cloud/library/golang-common.git"
)

type UserService struct {
	Usersconn *grpc.ClientConn
	Userssvc  UsersClient
	Userreco  chan bool

	Id *lib.Identity
}

// InitUsers tst
func (s *UserService) InitUsers(usersHost string, tracer opentracing.Tracer, logger *logrus.Logger) chan bool {
	logentry := logrus.NewEntry(logger)
	logopts := []grpc_logrus.Option{
		grpc_logrus.WithDurationField(func(duration time.Duration) (key string, value interface{}) {
			return "grpc.time_ns", duration.Nanoseconds()
		}),
	}

	otopts := []grpc_opentracing.Option{
		grpc_opentracing.WithTracer(tracer),
	}

	var err error

	connect := make(chan bool)

	go func(lconn chan bool) {
		for {
			logrus.Info("Wait for connect")
			r := <-lconn
			logrus.WithFields(logrus.Fields{"reconn": r}).Info("conn chan receive")
			if r {
				for i := 1; i < 5; i++ {
					// s.usersconn, s.usersvc
					s.Usersconn, err = grpc.Dial(usersHost,
						grpc.WithInsecure(),
						grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
							grpc_logrus.UnaryClientInterceptor(logentry, logopts...),
							grpc_opentracing.UnaryClientInterceptor(otopts...),
							grpc_prometheus.UnaryClientInterceptor,
						)),
						grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
							grpc_logrus.StreamClientInterceptor(logentry, logopts...),
							grpc_opentracing.StreamClientInterceptor(otopts...),
							grpc_prometheus.StreamClientInterceptor,
						)),
					)
					if err != nil {
						logger.Fatalf("did not connect: %v, try : %d - sleep 5s", err, i)
						time.Sleep(2 * time.Second)
					} else {
						s.Userssvc = NewUsersClient(s.Usersconn)
						break
					}
				}
			} else {
				logrus.Info("end of goroutine - reconnect")
				return
			}
		}
	}(connect)

	logger.WithFields(logrus.Fields{"host": usersHost}).Info("Connexion au service gRPC 'Users'")
	connect <- true
	return connect
}

func (s *UserService) GetUser(ctx context.Context, usr *User) (*User, error) {
	for i := 0; i <= 5; i++ {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			logrus.Error("cannot get outgoing data")
		}
		logrus.WithFields(logrus.Fields{"usr": usr, "jwt": md.Get("authorization")}).Debug("get user")
		if usr == nil {
			logrus.Error("error get user, usr is nil")
		}
		if ctx == nil {
			logrus.Error("error get user, ctx is nil")
		}
		if md == nil {
			logrus.Error("error get user, md is nil")
		}
		if s.Userssvc == nil {
			logrus.Error("error get user, s.usersvc is nil")
			s.Userreco <- true
		}
		grp, err := s.Userssvc.Get(metadata.NewOutgoingContext(ctx, md), usr)
		logrus.WithFields(logrus.Fields{"ctx.err": ctx.Err(), "err": err}).Trace("error ctx get object")
		if err != nil {
			logrus.WithFields(logrus.Fields{"err": err}).Error("error get object")
			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Unavailable {
				s.Userreco <- true
			} else if errStatus.Code() == codes.Canceled {
				s.Userreco <- true
			} else if errStatus.Code() == codes.DeadlineExceeded {
				s.Userreco <- true
			} else if errStatus.Code() == codes.Aborted {
				s.Userreco <- true
			} else if errStatus.Code() == codes.Unauthenticated {
				logrus.WithFields(logrus.Fields{"jwt": md.Get("authorization")}).Info("ws-identity not identified")
				return nil, status.Error(codes.Unauthenticated, "unauthenticated")
			} else if errStatus.Code() == codes.InvalidArgument {
				return nil, status.Errorf(codes.InvalidArgument, "argument invalid %v", err)
			} else if errStatus.Code() == codes.NotFound {
				return nil, nil
			}
			// errStatus.Code() == codes.Internal = retry
		} else if ctx.Err() != nil {
			s.Userreco <- true
		} else {
			return grp, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "User not found")
}
