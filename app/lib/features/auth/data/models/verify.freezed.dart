// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'verify.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$VerifyRequest {

 String get code; String get email;
/// Create a copy of VerifyRequest
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$VerifyRequestCopyWith<VerifyRequest> get copyWith => _$VerifyRequestCopyWithImpl<VerifyRequest>(this as VerifyRequest, _$identity);

  /// Serializes this VerifyRequest to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is VerifyRequest&&(identical(other.code, code) || other.code == code)&&(identical(other.email, email) || other.email == email));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,code,email);

@override
String toString() {
  return 'VerifyRequest(code: $code, email: $email)';
}


}

/// @nodoc
abstract mixin class $VerifyRequestCopyWith<$Res>  {
  factory $VerifyRequestCopyWith(VerifyRequest value, $Res Function(VerifyRequest) _then) = _$VerifyRequestCopyWithImpl;
@useResult
$Res call({
 String code, String email
});




}
/// @nodoc
class _$VerifyRequestCopyWithImpl<$Res>
    implements $VerifyRequestCopyWith<$Res> {
  _$VerifyRequestCopyWithImpl(this._self, this._then);

  final VerifyRequest _self;
  final $Res Function(VerifyRequest) _then;

/// Create a copy of VerifyRequest
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? code = null,Object? email = null,}) {
  return _then(_self.copyWith(
code: null == code ? _self.code : code // ignore: cast_nullable_to_non_nullable
as String,email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [VerifyRequest].
extension VerifyRequestPatterns on VerifyRequest {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _VerifyRequest value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _VerifyRequest() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _VerifyRequest value)  $default,){
final _that = this;
switch (_that) {
case _VerifyRequest():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _VerifyRequest value)?  $default,){
final _that = this;
switch (_that) {
case _VerifyRequest() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String code,  String email)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _VerifyRequest() when $default != null:
return $default(_that.code,_that.email);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String code,  String email)  $default,) {final _that = this;
switch (_that) {
case _VerifyRequest():
return $default(_that.code,_that.email);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String code,  String email)?  $default,) {final _that = this;
switch (_that) {
case _VerifyRequest() when $default != null:
return $default(_that.code,_that.email);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _VerifyRequest implements VerifyRequest {
  const _VerifyRequest({required this.code, required this.email});
  factory _VerifyRequest.fromJson(Map<String, dynamic> json) => _$VerifyRequestFromJson(json);

@override final  String code;
@override final  String email;

/// Create a copy of VerifyRequest
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$VerifyRequestCopyWith<_VerifyRequest> get copyWith => __$VerifyRequestCopyWithImpl<_VerifyRequest>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$VerifyRequestToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _VerifyRequest&&(identical(other.code, code) || other.code == code)&&(identical(other.email, email) || other.email == email));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,code,email);

@override
String toString() {
  return 'VerifyRequest(code: $code, email: $email)';
}


}

/// @nodoc
abstract mixin class _$VerifyRequestCopyWith<$Res> implements $VerifyRequestCopyWith<$Res> {
  factory _$VerifyRequestCopyWith(_VerifyRequest value, $Res Function(_VerifyRequest) _then) = __$VerifyRequestCopyWithImpl;
@override @useResult
$Res call({
 String code, String email
});




}
/// @nodoc
class __$VerifyRequestCopyWithImpl<$Res>
    implements _$VerifyRequestCopyWith<$Res> {
  __$VerifyRequestCopyWithImpl(this._self, this._then);

  final _VerifyRequest _self;
  final $Res Function(_VerifyRequest) _then;

/// Create a copy of VerifyRequest
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? code = null,Object? email = null,}) {
  return _then(_VerifyRequest(
code: null == code ? _self.code : code // ignore: cast_nullable_to_non_nullable
as String,email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}


/// @nodoc
mixin _$VerifyResponse {

 TokenPair get data;
/// Create a copy of VerifyResponse
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$VerifyResponseCopyWith<VerifyResponse> get copyWith => _$VerifyResponseCopyWithImpl<VerifyResponse>(this as VerifyResponse, _$identity);

  /// Serializes this VerifyResponse to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is VerifyResponse&&(identical(other.data, data) || other.data == data));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,data);

@override
String toString() {
  return 'VerifyResponse(data: $data)';
}


}

/// @nodoc
abstract mixin class $VerifyResponseCopyWith<$Res>  {
  factory $VerifyResponseCopyWith(VerifyResponse value, $Res Function(VerifyResponse) _then) = _$VerifyResponseCopyWithImpl;
@useResult
$Res call({
 TokenPair data
});


$TokenPairCopyWith<$Res> get data;

}
/// @nodoc
class _$VerifyResponseCopyWithImpl<$Res>
    implements $VerifyResponseCopyWith<$Res> {
  _$VerifyResponseCopyWithImpl(this._self, this._then);

  final VerifyResponse _self;
  final $Res Function(VerifyResponse) _then;

/// Create a copy of VerifyResponse
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? data = null,}) {
  return _then(_self.copyWith(
data: null == data ? _self.data : data // ignore: cast_nullable_to_non_nullable
as TokenPair,
  ));
}
/// Create a copy of VerifyResponse
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TokenPairCopyWith<$Res> get data {
  
  return $TokenPairCopyWith<$Res>(_self.data, (value) {
    return _then(_self.copyWith(data: value));
  });
}
}


/// Adds pattern-matching-related methods to [VerifyResponse].
extension VerifyResponsePatterns on VerifyResponse {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _VerifyResponse value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _VerifyResponse() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _VerifyResponse value)  $default,){
final _that = this;
switch (_that) {
case _VerifyResponse():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _VerifyResponse value)?  $default,){
final _that = this;
switch (_that) {
case _VerifyResponse() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( TokenPair data)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _VerifyResponse() when $default != null:
return $default(_that.data);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( TokenPair data)  $default,) {final _that = this;
switch (_that) {
case _VerifyResponse():
return $default(_that.data);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( TokenPair data)?  $default,) {final _that = this;
switch (_that) {
case _VerifyResponse() when $default != null:
return $default(_that.data);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _VerifyResponse implements VerifyResponse {
  const _VerifyResponse({required this.data});
  factory _VerifyResponse.fromJson(Map<String, dynamic> json) => _$VerifyResponseFromJson(json);

@override final  TokenPair data;

/// Create a copy of VerifyResponse
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$VerifyResponseCopyWith<_VerifyResponse> get copyWith => __$VerifyResponseCopyWithImpl<_VerifyResponse>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$VerifyResponseToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _VerifyResponse&&(identical(other.data, data) || other.data == data));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,data);

@override
String toString() {
  return 'VerifyResponse(data: $data)';
}


}

/// @nodoc
abstract mixin class _$VerifyResponseCopyWith<$Res> implements $VerifyResponseCopyWith<$Res> {
  factory _$VerifyResponseCopyWith(_VerifyResponse value, $Res Function(_VerifyResponse) _then) = __$VerifyResponseCopyWithImpl;
@override @useResult
$Res call({
 TokenPair data
});


@override $TokenPairCopyWith<$Res> get data;

}
/// @nodoc
class __$VerifyResponseCopyWithImpl<$Res>
    implements _$VerifyResponseCopyWith<$Res> {
  __$VerifyResponseCopyWithImpl(this._self, this._then);

  final _VerifyResponse _self;
  final $Res Function(_VerifyResponse) _then;

/// Create a copy of VerifyResponse
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? data = null,}) {
  return _then(_VerifyResponse(
data: null == data ? _self.data : data // ignore: cast_nullable_to_non_nullable
as TokenPair,
  ));
}

/// Create a copy of VerifyResponse
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TokenPairCopyWith<$Res> get data {
  
  return $TokenPairCopyWith<$Res>(_self.data, (value) {
    return _then(_self.copyWith(data: value));
  });
}
}


/// @nodoc
mixin _$ResendOtpRequest {

 String get email;
/// Create a copy of ResendOtpRequest
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ResendOtpRequestCopyWith<ResendOtpRequest> get copyWith => _$ResendOtpRequestCopyWithImpl<ResendOtpRequest>(this as ResendOtpRequest, _$identity);

  /// Serializes this ResendOtpRequest to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ResendOtpRequest&&(identical(other.email, email) || other.email == email));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,email);

@override
String toString() {
  return 'ResendOtpRequest(email: $email)';
}


}

/// @nodoc
abstract mixin class $ResendOtpRequestCopyWith<$Res>  {
  factory $ResendOtpRequestCopyWith(ResendOtpRequest value, $Res Function(ResendOtpRequest) _then) = _$ResendOtpRequestCopyWithImpl;
@useResult
$Res call({
 String email
});




}
/// @nodoc
class _$ResendOtpRequestCopyWithImpl<$Res>
    implements $ResendOtpRequestCopyWith<$Res> {
  _$ResendOtpRequestCopyWithImpl(this._self, this._then);

  final ResendOtpRequest _self;
  final $Res Function(ResendOtpRequest) _then;

/// Create a copy of ResendOtpRequest
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? email = null,}) {
  return _then(_self.copyWith(
email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [ResendOtpRequest].
extension ResendOtpRequestPatterns on ResendOtpRequest {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ResendOtpRequest value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ResendOtpRequest() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ResendOtpRequest value)  $default,){
final _that = this;
switch (_that) {
case _ResendOtpRequest():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ResendOtpRequest value)?  $default,){
final _that = this;
switch (_that) {
case _ResendOtpRequest() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String email)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ResendOtpRequest() when $default != null:
return $default(_that.email);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String email)  $default,) {final _that = this;
switch (_that) {
case _ResendOtpRequest():
return $default(_that.email);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String email)?  $default,) {final _that = this;
switch (_that) {
case _ResendOtpRequest() when $default != null:
return $default(_that.email);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ResendOtpRequest implements ResendOtpRequest {
  const _ResendOtpRequest({required this.email});
  factory _ResendOtpRequest.fromJson(Map<String, dynamic> json) => _$ResendOtpRequestFromJson(json);

@override final  String email;

/// Create a copy of ResendOtpRequest
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ResendOtpRequestCopyWith<_ResendOtpRequest> get copyWith => __$ResendOtpRequestCopyWithImpl<_ResendOtpRequest>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ResendOtpRequestToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ResendOtpRequest&&(identical(other.email, email) || other.email == email));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,email);

@override
String toString() {
  return 'ResendOtpRequest(email: $email)';
}


}

/// @nodoc
abstract mixin class _$ResendOtpRequestCopyWith<$Res> implements $ResendOtpRequestCopyWith<$Res> {
  factory _$ResendOtpRequestCopyWith(_ResendOtpRequest value, $Res Function(_ResendOtpRequest) _then) = __$ResendOtpRequestCopyWithImpl;
@override @useResult
$Res call({
 String email
});




}
/// @nodoc
class __$ResendOtpRequestCopyWithImpl<$Res>
    implements _$ResendOtpRequestCopyWith<$Res> {
  __$ResendOtpRequestCopyWithImpl(this._self, this._then);

  final _ResendOtpRequest _self;
  final $Res Function(_ResendOtpRequest) _then;

/// Create a copy of ResendOtpRequest
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? email = null,}) {
  return _then(_ResendOtpRequest(
email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}

// dart format on
