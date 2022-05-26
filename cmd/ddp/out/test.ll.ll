; ModuleID = 'out/test.ll'
source_filename = "test.ddp"
target datalayout = "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"
target triple = "x86_64-pc-linux-gnu"

%ddpstring = type { i16*, i64 }

@0 = global [7 x i16] [i16 69, i16 114, i16 102, i16 111, i16 108, i16 103, i16 33]
@1 = global [5 x i16] [i16 116, i16 101, i16 115, i16 116, i16 49]
@2 = global [5 x i16] [i16 116, i16 101, i16 115, i16 116, i16 49]
@3 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@4 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]
@5 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@6 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]

declare %ddpstring* @inbuilt_string_from_constant(i16*, i64)

declare %ddpstring* @inbuilt_deep_copy_string(%ddpstring*)

declare void @inbuilt_decrement_ref_count(i8*)

declare void @inbuilt_increment_ref_count(i8*, i8)

define i64 @inbuilt_ddpmain() {
  %1 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([5 x i16], [5 x i16]* @1, i64 0, i64 0), i64 5)
  %2 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([5 x i16], [5 x i16]* @2, i64 0, i64 0), i64 5)
  %3 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %1, %ddpstring* %2)
  call void @inbuilt_Schreibe_Text(%ddpstring* %3)
  %4 = alloca i64, align 8
  br label %5

5:                                                ; preds = %7, %0
  %storemerge = phi i64 [ 0, %0 ], [ %9, %7 ]
  store i64 %storemerge, i64* %4, align 8
  %6 = icmp eq i64 %storemerge, 10000
  br i1 %6, label %15, label %10

7:                                                ; preds = %10
  %8 = load i64, i64* %4, align 8
  %9 = add i64 %8, 1
  br label %5

10:                                               ; preds = %5
  %11 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @3, i64 0, i64 0), i64 10)
  %12 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @4, i64 0, i64 0), i64 10)
  %13 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %11, %ddpstring* %12)
  %14 = bitcast %ddpstring* %13 to i8*
  call void @inbuilt_increment_ref_count(i8* %14, i8 0)
  %.cast = bitcast %ddpstring* %13 to i8*
  call void @inbuilt_decrement_ref_count(i8* %.cast)
  br label %7

15:                                               ; preds = %5
  %16 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @5, i64 0, i64 0), i64 10)
  %17 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @6, i64 0, i64 0), i64 10)
  %18 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %16, %ddpstring* %17)
  %19 = bitcast %ddpstring* %18 to i8*
  call void @inbuilt_increment_ref_count(i8* %19, i8 0)
  %.cast1 = bitcast %ddpstring* %18 to i8*
  call void @inbuilt_decrement_ref_count(i8* %.cast1)
  br label %20

20:                                               ; preds = %15
  ret i64 0
}

declare void @inbuilt_Schreibe_Buchstabe(i16)

declare void @inbuilt_Schreibe_Text(%ddpstring*)

define %ddpstring* @ddpfunc_string_test(%ddpstring* %s1, %ddpstring* %s2) {
  %1 = bitcast %ddpstring* %s1 to i8*
  call void @inbuilt_increment_ref_count(i8* %1, i8 0)
  %2 = alloca %ddpstring*, align 8
  %3 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s1)
  store %ddpstring* %3, %ddpstring** %2, align 8
  %4 = bitcast %ddpstring* %s1 to i8*
  call void @inbuilt_decrement_ref_count(i8* %4)
  %5 = bitcast %ddpstring* %s2 to i8*
  call void @inbuilt_increment_ref_count(i8* %5, i8 0)
  %6 = alloca %ddpstring*, align 8
  %7 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s2)
  store %ddpstring* %7, %ddpstring** %6, align 8
  %8 = bitcast %ddpstring* %s2 to i8*
  call void @inbuilt_decrement_ref_count(i8* %8)
  %9 = load %ddpstring*, %ddpstring** %2, align 8
  %10 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %9)
  call void @inbuilt_Schreibe_Text(%ddpstring* %10)
  call void @inbuilt_Schreibe_Buchstabe(i16 10)
  %11 = load %ddpstring*, %ddpstring** %6, align 8
  %12 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %11)
  call void @inbuilt_Schreibe_Text(%ddpstring* %12)
  call void @inbuilt_Schreibe_Buchstabe(i16 10)
  %13 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([7 x i16], [7 x i16]* @0, i64 0, i64 0), i64 7)
  %14 = bitcast %ddpstring* %13 to i8*
  call void @inbuilt_increment_ref_count(i8* %14, i8 0)
  %15 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %13)
  %16 = bitcast %ddpstring* %13 to i8*
  call void @inbuilt_decrement_ref_count(i8* %16)
  %17 = bitcast %ddpstring** %2 to i8**
  %18 = load i8*, i8** %17, align 8
  call void @inbuilt_decrement_ref_count(i8* %18)
  %19 = bitcast %ddpstring** %6 to i8**
  %20 = load i8*, i8** %19, align 8
  call void @inbuilt_decrement_ref_count(i8* %20)
  ret %ddpstring* %15
}
